package bulk_indexing

import (
	"fmt"
	"infini.sh/framework/core/config"
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
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
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
type BulkIndexingProcessor struct {
	bufferPool *bytebufferpool.Pool
	initLocker sync.RWMutex
	config     *Config
}

type Config struct {
	NumOfWorkers         int      `config:"worker_size"`
	IdleTimeoutInSecond  int      `config:"idle_timeout_in_seconds"`
	MaxConnectionPerHost int      `config:"max_connection_per_node"`
	BulkSizeInKb         int      `config:"bulk_size_in_kb,omitempty"`
	BulkSizeInMb         int      `config:"bulk_size_in_mb,omitempty"`
	Elasticsearch        string   `config:"elasticsearch,omitempty"`
	Level        string   `config:"level,omitempty"`

	Indices              []string `config:"index,omitempty"`
	EnabledShards        []string `config:"shards,omitempty"`
	Queues               []string `config:"queues,omitempty"`
	ValidateRequest      bool     `config:"valid_request"`

	RotateConfig rotate.RotateConfig          `config:"rotate"`
	BulkConfig   elastic2.BulkProcessorConfig `config:"bulk"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers:         1,
		MaxConnectionPerHost: 1,
		IdleTimeoutInSecond:  5,
		BulkSizeInMb:         10,
		ValidateRequest:      false,
		RotateConfig:         rotate.DefaultConfig,
		BulkConfig:         elastic2.DefaultBulkProcessorConfig,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := BulkIndexingProcessor{config: &cfg}
	return &runner, nil
}

func (processor *BulkIndexingProcessor) Name() string {
	return "bulk_indexing"
}

func (processor *BulkIndexingProcessor) Process(c *pipeline.Context) error {
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

	bulkSizeInByte := 1048576 * processor.config.BulkSizeInMb
	if processor.config.BulkSizeInKb > 0 {
		bulkSizeInByte = 1024 * processor.config.BulkSizeInKb
	}

	if processor.bufferPool == nil {
		processor.initLocker.Lock()
		if processor.bufferPool == nil {
			estimatedBulkSizeInByte := bulkSizeInByte + (bulkSizeInByte / 3)
			processor.bufferPool = bytebufferpool.NewPool(uint64(estimatedBulkSizeInByte), uint64(bulkSizeInByte*2))
		}
		processor.initLocker.Unlock()
	}

	//wait for nodes info
	var nodesFailureCount =0
	NODESINFO:

	meta := elastic.GetMetadata(processor.config.Elasticsearch)
	wg := sync.WaitGroup{}

	if meta == nil {
		return errors.New("metadata is nil")
	}


	if processor.config.Level=="cluster"{
		queueName := common.GetClusterLevelShuffleKey(processor.config.Elasticsearch)

		if global.Env().IsDebug {
			log.Trace("queueName:", queueName)
		}

		for i := 0; i < processor.config.NumOfWorkers; i++ {
			wg.Add(1)
			go processor.NewBulkWorker(c,bulkSizeInByte, &wg, queueName, meta.GetActiveHost())
		}
	}else{
		//index,shard,level
		if len(processor.config.Indices) > 0 {
			for _, v := range processor.config.Indices {
				indexSettings := meta.Indices[v]
				for i := 0; i < indexSettings.Shards; i++ {
					queueName := common.GetShardLevelShuffleKey(processor.config.Elasticsearch, v, i)
					shardInfo := meta.GetPrimaryShardInfo(v, i)

					if len(processor.config.EnabledShards) > 0 {
						if !util.ContainsAnyInArray(shardInfo.ShardID, processor.config.EnabledShards) {
							log.Debugf("%s-%s not enabled, skip processing", shardInfo.Index, shardInfo.ShardID)
							continue
						}
					}

					nodeInfo := meta.GetNodeInfo(shardInfo.NodeID)

					if global.Env().IsDebug {
						log.Debug(shardInfo.Index, ",", shardInfo.ShardID, ",", nodeInfo.Http.PublishAddress)
					}

					for i := 0; i < processor.config.NumOfWorkers; i++ {
						wg.Add(1)
						go processor.NewBulkWorker(c,bulkSizeInByte, &wg, queueName, nodeInfo.Http.PublishAddress)
					}
				}
			}
		} else { //node level


			if meta.Nodes == nil {
				nodesFailureCount++
				if nodesFailureCount>10{
					log.Debug("enough wait for none nil nodes")
					return errors.New("nodes is nil")
				}
				time.Sleep(10*time.Second)
				goto NODESINFO
			}

			//TODO only get data nodes or filtered nodes
			for k, v := range meta.Nodes {
				queueName := common.GetNodeLevelShuffleKey(processor.config.Elasticsearch, k)

				if global.Env().IsDebug {
					log.Trace("queueName:", queueName, ",", v)
					log.Debug("nodeInfo:", k, ",", v.Http.PublishAddress)
				}

				for i := 0; i < processor.config.NumOfWorkers; i++ {
					wg.Add(1)
					go processor.NewBulkWorker(c,bulkSizeInByte, &wg, queueName, v.Http.PublishAddress)
				}
			}
		}
	}

	if len(processor.config.Queues) > 0 {
		host := meta.GetActiveHost()
			for _, v := range processor.config.Queues {
				log.Debug("process bulk queue:", v)
				wg.Add(1)
				//TODO node.Http.PublishAddress 拿错地址，不可用怎么处理
				go processor.NewBulkWorker(c,bulkSizeInByte, &wg, v, host)
			}
	}

	wg.Wait()

	return nil
}

func (processor *BulkIndexingProcessor) NewBulkWorker(ctx *pipeline.Context,bulkSizeInByte int, wg *sync.WaitGroup, queueName string, host string) {

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
				ctx.Failed()
			}
		}
		wg.Done()
	}()

	log.Debug("start worker:", queueName, ", host:", host)

	mainBuf := processor.bufferPool.Get()
	mainBuf.Reset()
	defer processor.bufferPool.Put(mainBuf)

	idleDuration := time.Duration(processor.config.IdleTimeoutInSecond) * time.Second
	meta := elastic.GetMetadata(processor.config.Elasticsearch)

	bulkProcessor := elastic2.BulkProcessor{
		RotateConfig: processor.config.RotateConfig,
		Config:       processor.config.BulkConfig,
	}

	if bulkProcessor.Config.FailureRequestsQueue == "" {
		bulkProcessor.Config.FailureRequestsQueue = fmt.Sprintf("%v-failure", processor.config.Elasticsearch)
	}
	if bulkProcessor.Config.DeadletterRequestsQueue == "" {
		bulkProcessor.Config.DeadletterRequestsQueue = fmt.Sprintf("%v-dead_letter", processor.config.Elasticsearch)
	}

	if bulkProcessor.Config.InvalidRequestsQueue == "" {
		bulkProcessor.Config.InvalidRequestsQueue = fmt.Sprintf("%v-invalid", processor.config.Elasticsearch)
	}

	var lastCommit time.Time=time.Now()

READ_DOCS:
	for {
		if ctx.IsCanceled(){
			goto CLEAN_BUFFER
		}

		//each message is complete bulk message, must be end with \n
		pop, _, err := queue.PopTimeout(queueName, idleDuration)
		if processor.config.ValidateRequest {
			common.ValidateBulkRequest("write_pop", string(pop))
		}

		if err != nil {
			panic(err)
		}

		if len(pop) > 0 {
			stats.IncrementBy("elasticsearch."+processor.config.Elasticsearch+".bulk", "bytes.received", int64(mainBuf.Len()))
			mainBuf.Write(pop)
		}

		if time.Since(lastCommit)>idleDuration && mainBuf.Len()>0{
			if global.Env().IsDebug {
				log.Trace("hit idle timeout, ", idleDuration.String())
			}
			goto CLEAN_BUFFER
		}

		if mainBuf.Len() > (bulkSizeInByte) {
			if global.Env().IsDebug {
				log.Trace("hit buffer size,", mainBuf.Len(), ", ", queueName, ", submit")
			}
			goto CLEAN_BUFFER
		}

	}

CLEAN_BUFFER:

	lastCommit=time.Now()

	if mainBuf.Len() > 0 {

		start := time.Now()
		data := mainBuf.Bytes()
		log.Trace(meta.Config.Name, ", starting submit bulk request")
		status, success := bulkProcessor.Bulk(meta, host, data)
		stats.Timing("elasticsearch."+meta.Config.Name+".bulk", "elapsed_ms", time.Since(start).Milliseconds())
		log.Debug(meta.Config.Name,", ", host, ", result:", success, ", status:", status, ", size:", util.ByteSize(uint64(mainBuf.Len())), ", elapsed:", time.Since(start))

		switch success {
		case elastic2.SUCCESS:
			stats.IncrementBy("elasticsearch."+processor.config.Elasticsearch+".bulk", "bytes.success", int64(mainBuf.Len()))
			break
		case elastic2.INVALID:
			stats.IncrementBy("elasticsearch."+processor.config.Elasticsearch+".bulk", "bytes.invalid", int64(mainBuf.Len()))
			break
		case elastic2.PARTIAL:
			stats.IncrementBy("elasticsearch."+processor.config.Elasticsearch+".bulk", "bytes.partial", int64(mainBuf.Len()))
			break
		case elastic2.FAILURE:
			stats.IncrementBy("elasticsearch."+processor.config.Elasticsearch+".bulk", "bytes.failure", int64(mainBuf.Len()))
			break
		}

		mainBuf.Reset()
	}

	if ctx.IsCanceled(){
		return
	}

	goto READ_DOCS

}
