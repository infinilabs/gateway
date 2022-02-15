package bulk_indexing

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/gateway/common"
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
	bufferPool     *bytebufferpool.Pool
	config         *Config
	runningConfigs map[string]*queue.Config
	bulkSizeInByte int
	wg             sync.WaitGroup
	inFlightQueueConfigs sync.Map
	detectorRunning bool
	id string
}

type Config struct {
	NumOfWorkers         int    `config:"worker_size"`

	IdleTimeoutInSecond  int    `config:"idle_timeout_in_seconds"`
	MaxConnectionPerHost int    `config:"max_connection_per_node"`

	BulkSizeInKb         int    `config:"bulk_size_in_kb,omitempty"`
	BulkSizeInMb         int    `config:"bulk_size_in_mb,omitempty"`

	Queues          map[string]interface{} `config:"queues,omitempty"`

	FetchMinBytes    int `config:"fetch_min_bytes"`
	FetchMaxBytes    int `config:"fetch_max_bytes"`
	FetchMaxMessages int `config:"fetch_max_messages"`
	FetchMaxWaitMs   int `config:"fetch_max_wait_ms"`

	MaxWorkers int      `config:"max_worker_size"`

	DetectActiveQueue bool     `config:"detect_active_queue"`
	DetectIntervalInMs   int         `config:"detect_interval"`

	ValidateRequest bool     `config:"valid_request"`
	SkipEmptyQueue bool     `config:"skip_empty_queue"`
	SkipOnMissingInfo bool  `config:"skip_info_missing"`

	RotateConfig rotate.RotateConfig          `config:"rotate"`
	BulkConfig   elastic2.BulkProcessorConfig `config:"bulk"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers:         1,
		MaxWorkers:           10,
		MaxConnectionPerHost: 1,
		IdleTimeoutInSecond:  5,
		BulkSizeInMb:         10,
		DetectIntervalInMs:   10000,
		Queues: map[string]interface{}{},

		FetchMinBytes:   1,
		FetchMaxMessages:   100,
		FetchMaxWaitMs:   10000,

		DetectActiveQueue:    true,
		ValidateRequest:      false,
		SkipEmptyQueue:      true,
		SkipOnMissingInfo:   false,
		RotateConfig:         rotate.DefaultConfig,
		BulkConfig:           elastic2.DefaultBulkProcessorConfig,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := BulkIndexingProcessor{
		id:util.GetUUID(),
		config: &cfg,
		runningConfigs: map[string]*queue.Config{},
		inFlightQueueConfigs:sync.Map{},
	}

	runner.bulkSizeInByte= 1048576 * runner.config.BulkSizeInMb
	if runner.config.BulkSizeInKb > 0 {
		runner.bulkSizeInByte = 1024 * runner.config.BulkSizeInKb
	}

	estimatedBulkSizeInByte := runner.bulkSizeInByte + (runner.bulkSizeInByte / 3)
	runner.bufferPool = bytebufferpool.NewPool(uint64(estimatedBulkSizeInByte), uint64(runner.bulkSizeInByte*2))

	runner.wg = sync.WaitGroup{}

	return &runner, nil
}

func (processor *BulkIndexingProcessor) Name() string {
	return "bulk_indexing"
}

func (processor *BulkIndexingProcessor) Process(c *pipeline.Context) error {

	defer processor.wg.Wait()

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
				log.Error("error in bulk indexing processor,", v)
			}
		}
		log.Trace("exit bulk indexing processor")
	}()

	//handle updates
	if processor.config.DetectActiveQueue{
		log.Tracef("detectorRunning [%v]",processor.detectorRunning)
		if !processor.detectorRunning{
			processor.detectorRunning=true
			go func(c *pipeline.Context) {
				log.Tracef("[%v] init detector for active queue",processor.id)

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
							log.Error("error in bulk indexing processor,", v)
						}
					}
					processor.detectorRunning=false
					log.Debug("exit detector for active queue")
				}()

				for {
					log.Tracef("inflight queues: %v",util.MapLength(&processor.inFlightQueueConfigs))

					if global.Env().IsDebug{
						processor.inFlightQueueConfigs.Range(func(key, value interface{}) bool {
							log.Tracef("inflight queue:%v",key)
							return true
						})
					}

					cfgs:=queue.GetQueuesFilterByLabel(processor.config.Queues)
					for _,v:=range cfgs{
						if c.IsCanceled() {
							return
						}
						//if have depth and not in in flight
						if queue.Depth(v)>0{
							_,ok:=processor.inFlightQueueConfigs.Load(v.Id)
							if !ok{
								log.Tracef("detecting new queue: %v",v.Name)
								processor.HandleQueueConfig(v,c)
							}
						}
					}
					if processor.config.DetectIntervalInMs>0{
						time.Sleep(time.Millisecond*time.Duration(processor.config.DetectIntervalInMs))
					}
				}
			}(c)
		}
	}else{
		cfgs:=queue.GetQueuesFilterByLabel(processor.config.Queues)
		log.Debugf("filter queue by:%v, num of queues:%v",processor.config.Queues,len(cfgs))
		for _,v:=range cfgs{
			log.Tracef("checking queue: %v",v)
			processor.HandleQueueConfig(v,c)
		}
	}

	return nil
}

func (processor *BulkIndexingProcessor) HandleQueueConfig(v *queue.Config,c *pipeline.Context){

	if processor.config.SkipEmptyQueue{
		if queue.Depth(v)<=0{
			if global.Env().IsDebug{
				log.Tracef("skip empty queue:[%v]",v.Name)
			}
			return
		}
	}

	elasticsearch,ok:=v.Labels["elasticsearch"]
	if !ok{
		log.Errorf("label [elasticsearch] was not found in: %v",v)
		return
	}

	meta := elastic.GetMetadata(util.ToString(elasticsearch))
	if meta == nil {
		log.Debugf("metadata for [%v] is nil",elasticsearch)
		return
	}

	level,ok:=v.Labels["level"]

	if level=="node"{
		nodeID,ok:=v.Labels["node_id"]
		if ok{
			nodeInfo := meta.GetNodeInfo(util.ToString(nodeID))
			if nodeInfo!=nil{
				host:=nodeInfo.GetHttpPublishHost()
				processor.wg.Add(1)
				go processor.NewBulkWorker("242",c, processor.bulkSizeInByte, v, host)
				return
			}else{
				log.Debugf("node info not found: %v",nodeID)
			}
		}else{
			log.Debugf("node_id not found: %v",v)
		}
	}else if level=="shard"||level=="partition"{
		index,ok:=v.Labels["index"]
		if ok{
			routingTable, err := meta.GetIndexRoutingTable(util.ToString(index))
			if err != nil {
				log.Error(err)
				return
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
									nodeHost:=nodeInfo.GetHttpPublishHost()
									processor.wg.Add(1)
									go processor.NewBulkWorker("270",c, processor.bulkSizeInByte, v, nodeHost)
									return
								}else{
									log.Debugf("nodeInfo not found: %v",v)
								}
							}else{
								log.Debugf("nodeID not found: %v",v)
							}
							if processor.config.SkipOnMissingInfo{
								return
							}
						}
					}
				}else{
					log.Debugf("routing table not found: %v",v)
				}
			}else{
				log.Debugf("shard not found: %v",v)
			}
		}else{
			log.Debugf("index not found: %v",v)
		}
		if processor.config.SkipOnMissingInfo{
			return
		}
	}

	host := meta.GetActiveHost()
	log.Debugf("random choose node [%v] to consume queue [%v]",host,v.Id)
	processor.wg.Add(1)
	go processor.NewBulkWorker("300",c, processor.bulkSizeInByte, v, host)
}

func (processor *BulkIndexingProcessor) NewBulkWorker(tag string ,ctx *pipeline.Context, bulkSizeInByte int, qConfig *queue.Config, host string) {

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
				log.Error("error in bulk indexing processor,", v)
			}
		}
		processor.wg.Done()
		log.Trace("exit bulk indexing processor")
	}()


	key:=fmt.Sprintf("%v",qConfig.Id)

	if processor.config.MaxWorkers>0&&util.MapLength(&processor.inFlightQueueConfigs)>processor.config.MaxWorkers{
		log.Debugf("reached max num of workers, skip init [%v]",qConfig.Name)
		return
	}

	var workerID=util.GetUUID()
	_,exists:= processor.inFlightQueueConfigs.Load(key)
	if exists{
		log.Errorf("[%v], queue [%v] has more then one consumer",tag,qConfig.Id)
		return
	}

	processor.inFlightQueueConfigs.Store(key,workerID)
	log.Debugf("starting worker:[%v], queue:[%v], host:[%v]",workerID, qConfig.Name, host)

	mainBuf := processor.bufferPool.Get()
	defer processor.bufferPool.Put(mainBuf)
	var bulkProcessor elastic2.BulkProcessor
	var esClusterID string
	var meta *elastic.ElasticsearchMetadata
	var initOfffset string
	var offset string
	var consumer=queue.GetOrInitConsumerConfig(qConfig.Id,"group-001","consumer-001")

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
				log.Errorf("error in bulk_indexing worker[%v],queue:[%v],%v", workerID,qConfig.Id,v)
				ctx.Failed()
			}
		}

		processor.inFlightQueueConfigs.Delete(key)

		//cleanup buffer before exit worker
		continueNext :=processor.submitBulkRequest(esClusterID,meta,host,bulkProcessor,mainBuf)
		if continueNext {
			if offset!=""&&initOfffset!=offset{
				ok,err:=queue.CommitOffset(qConfig,consumer,offset)
				if !ok||err!=nil{
					panic(err)
				}
			}
		}else{
			log.Errorf("error between offset [%v]-[%v]",initOfffset,offset)
		}
		log.Debugf("exit worker[%v], queue:[%v]",workerID,qConfig.Id)
	}()


	idleDuration := time.Duration(processor.config.IdleTimeoutInSecond) * time.Second
	elasticsearch,ok:=qConfig.Labels["elasticsearch"]
	if !ok{
		panic(errors.Errorf("label [elasticsearch] was not found: %v", qConfig))
	}
	esClusterID=util.ToString(elasticsearch)
	meta = elastic.GetMetadata(esClusterID)
	if meta == nil {
		panic(errors.Errorf("cluster metadata [%v] not ready", esClusterID))
	}

	bulkProcessor = elastic2.BulkProcessor{
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
	initOfffset,_=queue.GetOffset(qConfig,consumer)
	offset=initOfffset

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
		//pop, timeout, err := queue.PopTimeout(qConfig, idleDuration)

		log.Debugf("worker:[%v] start consume queue:[%v] offset:%v",workerID,qConfig.Id,offset)

		ctx1,messages,timeout,err:=queue.Consume(qConfig,consumer.Name,offset,processor.config.FetchMaxMessages,time.Millisecond*time.Duration(processor.config.FetchMaxWaitMs))
		if global.Env().IsDebug{
			log.Debugf("[%v] consume message:%v,offset:%v,next:%v,timeout:%v,err:%v",consumer.Name,len(messages),ctx1.InitOffset,ctx1.NextOffset,timeout,err)
		}

		if timeout{
			log.Tracef("timeout on queue:[%v]",qConfig.Name)
			ctx.Failed()
			goto CLEAN_BUFFER
		}

		if err != nil {
			log.Tracef("error on queue:[%v]",qConfig.Name)
			if err.Error()=="EOF" {
				if len(messages)>0{
					goto HANDLE_MESSAGE
				}
				return
			}
			panic(err)
		}

		HANDLE_MESSAGE:

		//update temp offset, not committed, continued reading
		offset=ctx1.NextOffset

		if len(messages) > 0 {
			for _,pop:=range messages{

				if processor.config.ValidateRequest {
					common.ValidateBulkRequest("write_pop", string(pop.Data))
				}

				stats.IncrementBy("elasticsearch."+esClusterID+".bulk", "bytes_received_from_queue", int64(mainBuf.Len()))

				mainBuf.Write(pop.Data)

				if global.Env().IsDebug {
					log.Tracef("message size: %v", util.ByteSize(uint64(mainBuf.Len())))
				}

				if mainBuf.Len() > (bulkSizeInByte) {
					if global.Env().IsDebug {
						log.Trace("hit buffer size,", mainBuf.Len(), ", ", qConfig.Name, ", submit")
					}
					//submit request
					processor.submitBulkRequest(esClusterID,meta,host,bulkProcessor,mainBuf)
				}

			}
		}

		if time.Since(lastCommit) > idleDuration && mainBuf.Len() > 0 {
			if global.Env().IsDebug {
				log.Trace("hit idle timeout, ", idleDuration.String())
			}
			goto CLEAN_BUFFER
		}

	}

CLEAN_BUFFER:

	lastCommit = time.Now()
	//TODO, check bulk result, if ok, then commit offset, or retry non-200 requests, or save failure offset
	continueNext:=processor.submitBulkRequest(esClusterID,meta,host,bulkProcessor,mainBuf)
	if continueNext{
		if offset!=""{
			ok,err:=queue.CommitOffset(qConfig,consumer,offset)
			if !ok||err!=nil{
				panic(err)
			}
		}
	}else{
		//logging failure offset boundry
		log.Errorf("error between offset [%v]-[%v]",initOfffset,offset)
	}

	if offset==""||ctx.IsCanceled()||ctx.IsFailed() {
		log.Tracef("invalid offset or canceled, return on queue:[%v]",qConfig.Name)
		return
	}

	log.Tracef("goto READ_DOCS, return on queue:[%v]",qConfig.Name)

	goto READ_DOCS

}

func (processor *BulkIndexingProcessor) submitBulkRequest(esClusterID string, meta *elastic.ElasticsearchMetadata, host string, bulkProcessor elastic2.BulkProcessor, mainBuf *bytebufferpool.ByteBuffer)bool {

	log.Tracef("submit BulkRequest")

	if  mainBuf==nil||meta==nil{
		return true
	}

	if mainBuf.Len() > 0 {

		log.Trace(meta.Config.Name, ", starting submit bulk request")

		start := time.Now()
		data := mainBuf.Bytes()
		contrinueRequest,status, success, err := bulkProcessor.Bulk(meta, host, data)
		stats.Timing("elasticsearch."+esClusterID+".bulk", "elapsed_ms", time.Since(start).Milliseconds())
		log.Debug(meta.Config.Name, ", ", host, ", result:", success, ", status:", status, ", size:", util.ByteSize(uint64(mainBuf.Len())), ", elapsed:", time.Since(start)," ",err)
		stats.IncrementBy("elasticsearch."+esClusterID+".bulk", string(success+".bytes"), int64(mainBuf.Len()))
		stats.IncrementBy("elasticsearch."+esClusterID+".bulk", fmt.Sprintf("%v.bytes",status), int64(mainBuf.Len()))
		mainBuf.Reset()
		return contrinueRequest
	}

	return true
}
