package indexing

import (
	"crypto/tls"
	"fmt"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/lib/bytebufferpool"
	elastic2 "infini.sh/gateway/proxy/filters/elastic"
	"runtime"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

//#操作合并任务
//写入到本地一个队列，hash 散列
//【内存、磁盘、Kafka 三种持久化选项】以分片为单位合并数据，写本地磁盘队列，一个分片一个队列
//读取分片数据，发送到所在节点上

//#bulk 发送任务
//以节点为单位，然后以主分片为单位进行流量合并发送
//一个节点一个 go 协程，用于发送数据

//#将写模式改成拉模式，由各个分片主动去拉数据
//各个分片的数据线本地压缩好，变成固定大小的包
//由各个节点所在的 agent，压缩传输过去之后，本地快速重建
//调用目标节点所在的 agent 服务，rpc 远程写磁盘数据，然后目标服务器本地读取磁盘队列。

//读各个分片的数据，写 es

//处理 bulk 格式的数据索引。
type BulkIndexingJoint struct {
	param.Parameters
	bufferPool *bytebufferpool.Pool
	initLocker sync.RWMutex
}

func (joint BulkIndexingJoint) Name() string {
	return "bulk_indexing"
}



func (joint BulkIndexingJoint) Process(c *pipeline.Context) error {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error in bulk indexer,", v)
			}
		}
	}()

	workers, _ := joint.GetInt("worker_size", 1)
	elasticsearch := joint.GetStringOrDefault("elasticsearch", "default")
	enabledShards, checkShards := joint.GetStringArray("shards")

	bulkSizeInKB, _ := joint.GetInt("bulk_size_in_kb", 0)
	bulkSizeInMB, _ := joint.GetInt("bulk_size_in_mb", 10)
	bulkSizeInByte := 1048576 * bulkSizeInMB
	if bulkSizeInKB > 0 {
		bulkSizeInByte = 1024 * bulkSizeInKB
	}

	if joint.bufferPool==nil{
		joint.initLocker.Lock()
		if joint.bufferPool==nil{
			estimatedBulkSizeInByte:=bulkSizeInByte+(bulkSizeInByte/3)
			joint.bufferPool=bytebufferpool.NewPool(uint64(estimatedBulkSizeInByte),uint64(bulkSizeInByte*2))
		}
		joint.initLocker.Unlock()
	}


	meta := elastic.GetMetadata(elasticsearch)
	wg := sync.WaitGroup{}

	if meta == nil {
		return errors.New("metadata is nil")
	}

	esInstanceVal := joint.MustGetString("elasticsearch")
	indices, isIndex := joint.GetStringArray("index")

	//index,shard,level
	if isIndex {
		for _, v := range indices {
			indexSettings := meta.Indices[v]
			for i := 0; i < indexSettings.Shards; i++ {
				queueName := common.GetShardLevelShuffleKey(esInstanceVal, v, i)
				shardInfo := meta.GetPrimaryShardInfo(v, i)

				if checkShards && len(enabledShards) > 0 {
					if !util.ContainsAnyInArray(shardInfo.ShardID, enabledShards) {
						log.Debugf("%s-%s not enabled, skip processing", shardInfo.Index, shardInfo.ShardID)
						continue
					}
				}

				nodeInfo := meta.GetNodeInfo(shardInfo.NodeID)

				if global.Env().IsDebug {
					log.Debug(shardInfo.Index, ",", shardInfo.ShardID, ",", nodeInfo.Http.PublishAddress)
				}

				for i := 0; i < workers; i++ {
					wg.Add(1)
					go joint.NewBulkWorker(bulkSizeInByte, &wg, queueName, nodeInfo.Http.PublishAddress)
				}
			}
		}
	} else { //node level
		if meta.Nodes == nil {
			return errors.New("nodes is nil")
		}

		//TODO only get data nodes or filtred nodes
		for k, v := range meta.Nodes {
			queueName := common.GetNodeLevelShuffleKey(esInstanceVal, k)

			if global.Env().IsDebug {
				log.Trace("queueName:", queueName, ",", v)
				log.Debug("nodeInfo:", k, ",", v.Http.PublishAddress)
			}

			for i := 0; i < workers; i++ {
				wg.Add(1)
				go joint.NewBulkWorker(bulkSizeInByte, &wg, queueName, v.Http.PublishAddress)
			}
		}
	}

	//start deadline ingest
	if joint.GetBool("process_dead_letter_queue", false) {
		deadLetterQueueName := joint.GetStringOrDefault("dead_letter_queue", fmt.Sprintf("%v-failed_bulk_messages", esInstanceVal))
		v := meta.GetActiveNodeInfo()
		go joint.NewBulkWorker(bulkSizeInByte, &wg, deadLetterQueueName, v.Http.PublishAddress)
	}

	wg.Wait()

	return nil
}


func (joint BulkIndexingJoint) NewBulkWorker(bulkSizeInByte int, wg *sync.WaitGroup, queueName string, endpoint string) {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error in indexer,", v)
				wg.Done()
			}
		}
	}()

	log.Debug("start worker:", queueName, ", endpoint:", endpoint)

	mainBuf := joint.bufferPool.Get()
	mainBuf.Reset()
	defer joint.bufferPool.Put(mainBuf)
	esInstanceVal := joint.MustGetString("elasticsearch")
	validateRequest := joint.GetBool("valid_request", false)
	deadLetterQueueName := joint.GetStringOrDefault("dead_letter_queue", fmt.Sprintf("%v-failed_bulk_messages", esInstanceVal))
	timeOut := joint.GetIntOrDefault("idle_timeout_in_second", 5)
	connections := joint.GetIntOrDefault("max_connection_per_host", 1)

	idleDuration := time.Duration(timeOut) * time.Second
	cfg := elastic.GetConfig(esInstanceVal)

	bulkProcessor := elastic2.BulkProcessor{
		RotateConfig: rotate.RotateConfig{
			Compress:     joint.GetBool("compress_after_rotate", true),
			MaxFileAge:   joint.GetIntOrDefault("max_file_age", 0),
			MaxFileCount: joint.GetIntOrDefault("max_file_count", 100),
			MaxFileSize:  joint.GetIntOrDefault("max_file_size_in_mb", 1024),
		},
		Compress:                  joint.GetBool("compress", true),
		Log400Message:             joint.GetBool("log_400_message", true),
		LogInvalidMessage:         joint.GetBool("log_invalid_message", true),
		LogInvalid200Message:      joint.GetBool("log_invalid_200_message", true),
		LogInvalid200RetryMessage: joint.GetBool("log_200_retry_message", true),
		Log429RetryMessage:        joint.GetBool("log_429_retry_message", true),
		RetryDelayInSeconds:       joint.GetIntOrDefault("retry_delay_in_second", 1),
		RejectDelayInSeconds:      joint.GetIntOrDefault("reject_retry_delay_in_second", 1),
		MaxRejectRetryTimes:       joint.GetIntOrDefault("max_reject_retry_times", 3),
		MaxRetryTimes:             joint.GetIntOrDefault("max_retry_times", 3),
		MaxRequestBodySize:        joint.GetIntOrDefault("max_logged_request_body_size", 1024),
		MaxResponseBodySize:       joint.GetIntOrDefault("max_logged_response_body_size", 1024),
	}

	httpClient := fasthttp.Client{
		MaxConnsPerHost:     connections,
		MaxConnDuration:     0,
		MaxIdleConnDuration: 0,
		ReadTimeout:         time.Second * 60,
		WriteTimeout:        time.Second * 60,
		TLSConfig:           &tls.Config{InsecureSkipVerify: true},
	}

READ_DOCS:
	for {
		//each message is complete bulk message, must be end with \n
		pop, ok, err := queue.PopTimeout(queueName, idleDuration)
		if validateRequest {
			common.ValidateBulkRequest("write_pop", string(pop))
		}

		if err != nil {
			panic(err)
		}

		if ok {
			if global.Env().IsDebug {
				log.Tracef("%v no message input: %v", idleDuration, queueName)
			}
			goto CLEAN_BUFFER
		}

		if len(pop) > 0 {
			//log.Info("received message,",util.SubString(string(pop),0,100))
			stats.IncrementBy("bulk", "bytes_received", int64(mainBuf.Len()))
			mainBuf.Write(pop)
		}

		if mainBuf.Len() > (bulkSizeInByte) {
			if global.Env().IsDebug {
				log.Trace("hit buffer size,", mainBuf.Len(), ", ", queueName, ", submit")
			}
			goto CLEAN_BUFFER
		} else {
			//fmt.Println("size too small",mainBuf.Len(),"vs",bulkSizeInByte)
		}
	}

CLEAN_BUFFER:

	if mainBuf.Len() > 0 {

		start := time.Now()
		data := mainBuf.Bytes()
		log.Trace(cfg.Name, ", starting submit bulk request")

		status, success := bulkProcessor.Bulk(cfg, endpoint, data, &httpClient)
		log.Debug(cfg.Name, ", success:", success, ", status:", status, ", size:", util.ByteSize(uint64(mainBuf.Len())), ", elapsed:", time.Since(start))

		if !success {
			err := queue.Push(deadLetterQueueName, data)
			if err != nil {
				panic(err)
			}
			if global.Env().IsDebug {
				log.Warn("re-enqueue bulk messages to dead_letter queue")
			}
			stats.IncrementBy("bulk", "bytes_processed_failed", int64(mainBuf.Len()))
		} else {
			stats.IncrementBy("bulk", "bytes_processed_success", int64(mainBuf.Len()))
		}

		mainBuf.Reset()
		//TODO handle retry and fallback/over, dead letter queue
		//set services to failure, need manual restart
		//process dead letter queue first next round

	} else {
		//fmt.Println("timeout but CLEAN_BUFFER", mainBuf.Len())
	}

	goto READ_DOCS

}
