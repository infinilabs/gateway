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
	runningConfigs map[string]*queue.Config
}

type Config struct {
	NumOfWorkers         int    `config:"worker_size"`
	IdleTimeoutInSecond  int    `config:"idle_timeout_in_seconds"`
	MaxConnectionPerHost int    `config:"max_connection_per_node"`
	BulkSizeInKb         int    `config:"bulk_size_in_kb,omitempty"`
	BulkSizeInMb         int    `config:"bulk_size_in_mb,omitempty"`

	Queues          map[string]interface{} `config:"queues,omitempty"`

	ValidateRequest bool     `config:"valid_request"`

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
		BulkConfig:           elastic2.DefaultBulkProcessorConfig,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := BulkIndexingProcessor{config: &cfg,runningConfigs: map[string]*queue.Config{}}

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

	wg := sync.WaitGroup{}

	cfgs:=queue.GetQueuesFilterByLabel(processor.config.Queues)

	log.Debugf("filter queue by:%v, num of queues:%v",processor.config.Queues,len(cfgs))

	for _,v:=range cfgs{
		elasticsearch,ok:=v.Labels["elasticsearch"]
		if !ok{
			return errors.Errorf("label [elasticsearch] was not found in: %v",v)
		}

		meta := elastic.GetMetadata(util.ToString(elasticsearch))
		if meta == nil {
			return errors.Errorf("metadata for [%v] is nil",elasticsearch)
		}

		level,ok:=v.Labels["level"]
		host := meta.GetActiveHost()

		if ok{
			switch level {
			case "node": //node level
				nodeID,ok:=v.Labels["node_id"]
				if ok{
					nodeInfo := meta.GetNodeInfo(util.ToString(nodeID))
					if nodeInfo!=nil{
						host=nodeInfo.GetHttpPublishHost()
						wg.Add(1)
						go processor.NewBulkWorker(c, bulkSizeInByte, &wg, v, host)
					}
				}
				break
			case "shard": //shard level
				index,ok:=v.Labels["index"]
				if ok{
					routingTable, err := meta.GetIndexRoutingTable(util.ToString(index))
					if err != nil {
						return err
					}

					shard,ok:=v.Labels["shard"]
					if ok{
						shards,ok:=routingTable[util.ToString(shard)]
						if ok{
							for _,x:=range shards{
								if x.Primary{
									//each primary shard has a goroutine, or run by one goroutine
									if x.Node!=""{
										nodeInfo := meta.GetNodeInfo(x.Node)
										if nodeInfo!=nil{
											host=nodeInfo.GetHttpPublishHost()
											wg.Add(1)
											go processor.NewBulkWorker(c, bulkSizeInByte, &wg, v, host)
										}
									}
								}
							}
						}
					}
				}
				break
			case "partition":
				index,ok:=v.Labels["index"]
				if ok{
					shard,ok:=v.Labels["shard"]
					if ok{
						//partitionSize,ok1:=v.Labels["partition_size"]
						//partition,ok2:=v.Labels["partition"]
						//if ok1&&ok2{
							routingTable, err := meta.GetIndexRoutingTable(util.ToString(index))
							if err != nil {
								return err
							}
							shards,ok:=routingTable[util.ToString(shard)]
							if ok{
								for _,x:=range shards{
									if x.Primary{
										//each primary shard has a goroutine, or run by one goroutine
										if x.Node!=""{
											nodeInfo := meta.GetNodeInfo(x.Node)
											if nodeInfo!=nil{
												host=nodeInfo.GetHttpPublishHost()
												wg.Add(1)
												go processor.NewBulkWorker(c, bulkSizeInByte, &wg, v, host)
											}
										}
									}
								}
							}
						//}
					}
				}

				break
			default: //cluster level
				wg.Add(1)
				go processor.NewBulkWorker(c, bulkSizeInByte, &wg, v, host)
				break
			}
		}
	}

	wg.Wait()

	return nil
}

func (processor *BulkIndexingProcessor) NewBulkWorker(ctx *pipeline.Context, bulkSizeInByte int, wg *sync.WaitGroup, qConfig *queue.Config, host string) {

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

	log.Info("start worker:", qConfig.Name, ", host:", host)

	mainBuf := processor.bufferPool.Get()
	defer processor.bufferPool.Put(mainBuf)

	idleDuration := time.Duration(processor.config.IdleTimeoutInSecond) * time.Second
	elasticsearch,ok:=qConfig.Labels["elasticsearch"]
	if !ok{
		panic(errors.Errorf("label [elasticsearch] was not found: %v", qConfig))
	}
	esClusterID:=util.ToString(elasticsearch)
	meta := elastic.GetMetadata(esClusterID)

	if meta == nil {
		panic(errors.Errorf("cluster metadata [%v] not ready", esClusterID))
	}

	bulkProcessor := elastic2.BulkProcessor{
		RotateConfig: processor.config.RotateConfig,
		Config:       processor.config.BulkConfig,
	}

	if bulkProcessor.Config.FailureRequestsQueue == "" {
		bulkProcessor.Config.FailureRequestsQueue = fmt.Sprintf("%v-bulk-failure-items", esClusterID) //TODO record offset instead of new queue
	}
	if bulkProcessor.Config.DeadletterRequestsQueue == "" {
		bulkProcessor.Config.DeadletterRequestsQueue = fmt.Sprintf("%v-bulk-dead_letter-items", esClusterID)
	}

	if bulkProcessor.Config.InvalidRequestsQueue == "" {
		bulkProcessor.Config.InvalidRequestsQueue = fmt.Sprintf("%v-bulk-invalid-items", esClusterID)
	}
	if bulkProcessor.Config.PartialSuccessQueue == "" {
		bulkProcessor.Config.PartialSuccessQueue = fmt.Sprintf("%v-bulk-partial-success-items", esClusterID)
	}

	var lastCommit time.Time = time.Now()

READ_DOCS:
	for {
		if ctx.IsCanceled() {
			goto CLEAN_BUFFER
		}

		//TODO add config to enable check or not
		if !elastic.IsHostAvailable(host) {
			time.Sleep(time.Second * 1)
			log.Debugf("host [%v] is not available", host)
			goto READ_DOCS
		}

		//each message is complete bulk message, must be end with \n
		pop, _, err := queue.PopTimeout(qConfig, idleDuration)
		if processor.config.ValidateRequest {
			common.ValidateBulkRequest("write_pop", string(pop))
		}

		if err != nil {
			panic(err)
		}

		if len(pop) > 0 {
			stats.IncrementBy("elasticsearch."+esClusterID+".bulk", "bytes_received_from_queue", int64(mainBuf.Len()))
			mainBuf.Write(pop)
		}

		if time.Since(lastCommit) > idleDuration && mainBuf.Len() > 0 {
			if global.Env().IsDebug {
				log.Trace("hit idle timeout, ", idleDuration.String())
			}
			goto CLEAN_BUFFER
		}

		if mainBuf.Len() > (bulkSizeInByte) {
			if global.Env().IsDebug {
				log.Trace("hit buffer size,", mainBuf.Len(), ", ", qConfig.Name, ", submit")
			}
			goto CLEAN_BUFFER
		}

	}

CLEAN_BUFFER:

	lastCommit = time.Now()

	if mainBuf.Len() > 0 {

		start := time.Now()
		data := mainBuf.Bytes()
		log.Trace(meta.Config.Name, ", starting submit bulk request")
		status, success := bulkProcessor.Bulk(meta, host, data)
		stats.Timing("elasticsearch."+esClusterID+".bulk", "elapsed_ms", time.Since(start).Milliseconds())
		log.Debug(meta.Config.Name, ", ", host, ", result:", success, ", status:", status, ", size:", util.ByteSize(uint64(mainBuf.Len())), ", elapsed:", time.Since(start))
		stats.IncrementBy("elasticsearch."+esClusterID+".bulk", string(success+".bytes"), int64(mainBuf.Len()))
		stats.IncrementBy("elasticsearch."+esClusterID+".bulk", fmt.Sprintf("%v.bytes",status), int64(mainBuf.Len()))

		mainBuf.Reset()
	}

	if ctx.IsCanceled() {
		return
	}

	goto READ_DOCS

}
