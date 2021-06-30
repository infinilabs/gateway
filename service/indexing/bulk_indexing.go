package indexing

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
	"path"
	"runtime"
	"sync"
	"time"
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
	bulkSizeInKB, _ := joint.GetInt("bulk_size_in_kb", 0)
	bulkSizeInMB, _ := joint.GetInt("bulk_size_in_mb", 10)
	elasticsearch := joint.GetStringOrDefault("elasticsearch", "default")
	enabledShards,checkShards := joint.GetStringArray("shards")
	bulkSizeInByte:= 1048576 * bulkSizeInMB
	if bulkSizeInKB>0{
		bulkSizeInByte= 1024 * bulkSizeInKB
	}

	meta := elastic.GetMetadata(elasticsearch)
	wg := sync.WaitGroup{}

	if meta==nil{
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

				if checkShards && len(enabledShards)>0{
					if !util.ContainsAnyInArray(shardInfo.ShardID,enabledShards){
						log.Debugf("%s-%s not enabled, skip processing",shardInfo.Index,shardInfo.ShardID)
						continue
					}
				}

				nodeInfo := meta.GetNodeInfo(shardInfo.NodeID)

				if global.Env().IsDebug{
					log.Debug(shardInfo.Index,",",shardInfo.ShardID,",",nodeInfo.Http.PublishAddress)
				}

				for i := 0; i < workers; i++ {
					wg.Add(1)
					go joint.NewBulkWorker(bulkSizeInByte, &wg, queueName, func() string {
						return nodeInfo.Http.PublishAddress
					})
				}
			}
		}
	} else { //node level
		if meta.Nodes==nil{
			return errors.New("nodes is nil")
		}
		for k, v := range meta.Nodes {
			queueName := common.GetNodeLevelShuffleKey(esInstanceVal, k)

			if global.Env().IsDebug{
				log.Debug(k,",",v.Http.PublishAddress)
			}

			for i := 0; i < workers; i++ {
				wg.Add(1)
				go joint.NewBulkWorker(bulkSizeInByte, &wg, queueName, func() string {
					return v.Http.PublishAddress
				})
			}
		}
	}

	//start deadline ingest
	deadLetterQueueName := joint.GetStringOrDefault("dead_letter_queue","failed_bulk_messages")
	go joint.NewBulkWorker(bulkSizeInByte, &wg, deadLetterQueueName, func()string {
		v:= meta.GetActiveNodeInfo()
		if v!=nil{
			return v.Http.PublishAddress
		}
		panic("no endpoint found")
	})

	wg.Wait()

	return nil
}

func (joint BulkIndexingJoint) NewBulkWorker( bulkSizeInByte int, wg *sync.WaitGroup, queueName string, endpointFunc func()string) {
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

	endpoint:=endpointFunc()

	log.Debug("start worker:", queueName,", endpoint:",endpoint)

	mainBuf := bytes.Buffer{}
	esInstanceVal := joint.MustGetString("elasticsearch")
	validateRequest := joint.GetBool("valid_request",false)
	deadLetterQueueName := joint.GetStringOrDefault("dead_letter_queue","failed_bulk_messages")
	timeOut := joint.GetIntOrDefault("idle_timeout_in_second", 5)
	idleDuration := time.Duration(timeOut) * time.Second
	cfg := elastic.GetConfig(esInstanceVal)

	READ_DOCS:
	for {
		//each message is complete bulk message, must be end with \n
		pop, ok, err := queue.PopTimeout(queueName, idleDuration)
		if validateRequest{
			common.ValidateBulkRequest("write_pop",string(pop))
		}

		if err != nil {
			panic(err)
		}

		if ok {
			if global.Env().IsDebug {
				log.Tracef("%v no message input", idleDuration)
			}
			goto CLEAN_BUFFER
		}

		if len(pop)>0{
			//log.Info("received message,",util.SubString(string(pop),0,100))
			stats.IncrementBy("bulk", "bytes_received", int64(mainBuf.Len()))
			mainBuf.Write(pop)
		}

		if mainBuf.Len() > (bulkSizeInByte) {
			if global.Env().IsDebug {
				log.Trace("hit buffer size, ", mainBuf.Len())
			}
			goto CLEAN_BUFFER
		}else{
			//fmt.Println("size too small",mainBuf.Len(),"vs",bulkSizeInByte)
		}
	}

	CLEAN_BUFFER:

	if mainBuf.Len() > 0 {

		start:=time.Now()
		success:=joint.Bulk(cfg, endpoint, &mainBuf)
		log.Debug(cfg.Name,", bulk result:",success,", size:",util.ByteSize(uint64(mainBuf.Len())),", elpased:",time.Since(start))

		if !success{
			queue.Push(deadLetterQueueName,mainBuf.Bytes())
			if global.Env().IsDebug{
				log.Warn("re-enqueue bulk messages")
			}
			stats.IncrementBy("bulk", "bytes_processed_failed", int64(mainBuf.Len()))
		}else{
			stats.IncrementBy("bulk", "bytes_processed_success", int64(mainBuf.Len()))
		}

		mainBuf.Reset()
		//TODO handle retry and fallback/over, dead letter queue
		//set services to failure, need manual restart
		//process dead letter queue first next round

	}else{
		//fmt.Println("timeout but CLEAN_BUFFER", mainBuf.Len())
	}

	goto READ_DOCS

}

func (joint BulkIndexingJoint) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data *bytes.Buffer) bool{
	if data == nil || data.Len() == 0 {
		return true
	}

	if cfg.IsTLS() {
		endpoint = "https://" + endpoint
	} else {
		endpoint = "http://" + endpoint
	}
	url := fmt.Sprintf("%s/_bulk", endpoint)
	compress := joint.GetBool("compress",true)

	req := fasthttp.AcquireRequest()
	req.Reset()
	req.ResetBody()
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("bulk_indexing")

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	req.Header.SetContentType("application/x-ndjson")

	if cfg.BasicAuth != nil{
		req.URI().SetUsername(cfg.BasicAuth.Username)
		req.URI().SetPassword(cfg.BasicAuth.Password)
	}

	//set body
	if data.Len() > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data.Bytes(), fasthttp.CompressBestSpeed)
			if err != nil {
				return false
			}
		} else {
			req.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.Write(data.Bytes())
				w.Flush()
			})
		}
	}

	retryTimes:=1

DO:

	if cfg.TrafficControl!=nil{
	RetryRateLimit:

		if cfg.TrafficControl.MaxQpsPerNode>0{
			if !rate.GetRateLimiterPerSecond(cfg.Name,endpoint+"max_qps", int(cfg.TrafficControl.MaxQpsPerNode)).Allow(){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}

		if cfg.TrafficControl.MaxBytesPerNode>0{
			if !rate.GetRateLimiterPerSecond(cfg.Name,endpoint+"max_bps", int(cfg.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(),req.GetRequestLength()){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}

	}

	err := fastHttpClient.Do(req, resp)

	if err != nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return false
	}

	if resp == nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return false
	}

	// Do we need to decompress the response?
	var resbody =resp.GetRawBody()
	if global.Env().IsDebug{
		log.Trace(resp.StatusCode(),string(resbody))
	}

	if resp.StatusCode()==400{

		if joint.GetBool("log_bulk_message",true) {
			path1 := path.Join(global.Env().GetWorkingDir(), "bulk_400_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("\nURL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody:=data.Bytes()
			resBody1:=resbody
			if truncateSize>0{
				if len(reqBody)>truncateSize{
					reqBody=reqBody[:truncateSize]
				}
				if len(resBody1)>truncateSize{
					resBody1=resBody1[:truncateSize]
				}
			}
			util.FileAppendNewLineWithByte(path1,util.EscapeNewLine(reqBody) )
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)

			//requestBody:=data.String()
			//stringLines:=strings.Split(requestBody,"\n")
			//for _,v:=range stringLines{
			//	obj:=map[string]interface{}{}
			//	err:=util.FromJSONBytes([]byte(v),&obj)
			//	if err!=nil{
			//		log.Error("invalid json,",util.SubString(v,0,512),err)
			//		break
			//	}
			//}


		}
		return false
	}

	//TODO check resp body's error
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		//200{"took":2,"errors":true,"items":[
			//handle error items
			//"errors":true
			hit:=util.LimitedBytesSearch(resbody,[]byte("\"errors\":true"),64)
			if hit{
				if joint.GetBool("log_bulk_message",true) {
					path1 := path.Join(global.Env().GetWorkingDir(), "bulk_req_failure.log")
					truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
					util.FileAppendNewLineWithByte(path1, []byte("\nURL:"))
					util.FileAppendNewLineWithByte(path1, []byte(url))
					util.FileAppendNewLineWithByte(path1, []byte("Request:"))
					reqBody:=data.Bytes()
					resBody1:=resbody
					if truncateSize>0{
						if len(reqBody)>truncateSize{
							reqBody=reqBody[:truncateSize]
						}
						if len(resBody1)>truncateSize{
							resBody1=resBody1[:truncateSize]
						}
					}
					util.FileAppendNewLineWithByte(path1,util.EscapeNewLine(reqBody))
					util.FileAppendNewLineWithByte(path1, []byte("Response:"))
					util.FileAppendNewLineWithByte(path1, resBody1)
				}
				if joint.GetBool("warm_retry_message",true){
					log.Warnf("elasticsearch bulk error, retried %v times, will try again",retryTimes)
				}

				if retryTimes>=joint.GetIntOrDefault("failed_retry_times", 3){
					if joint.GetBool("warm_retry_message",true){
						log.Errorf("elasticsearch failed, retried %v times, quit retry",retryTimes)
						log.Errorf(string(resbody))
					}
					return false
				}

				retryTimes++
				delayTime := joint.GetIntOrDefault("failed_retry_delay_in_second", 5)
				time.Sleep(time.Duration(delayTime)*time.Second)
				goto DO
			}

		return true
	} else if resp.StatusCode()==429 {
		log.Warnf("elasticsearch rejected, retried %v times, will try again",retryTimes)
		delayTime := joint.GetIntOrDefault("reject_retry_delay_in_second", 5)
		time.Sleep(time.Duration(delayTime)*time.Second)
		if retryTimes>=joint.GetIntOrDefault("reject_retry_times", 60){
			if joint.GetBool("warm_retry_message",true){
				log.Errorf("elasticsearch rejected, retried %v times, quit retry",retryTimes)
				log.Errorf(string(resbody))
			}
			return false
		}
		retryTimes++
		goto DO
	}else {
		if joint.GetBool("log_bulk_message",false){
			path1:=path.Join(global.Env().GetWorkingDir(),"bulk_error_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("\nURL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody:=data.Bytes()
			resBody1:=resbody
			if truncateSize>0{
				if len(reqBody)>truncateSize{
					reqBody=reqBody[:truncateSize-1]
				}
				if len(resBody1)>truncateSize{
					resBody1=resBody1[:truncateSize-1]
				}
			}
			util.FileAppendNewLineWithByte(path1,util.EscapeNewLine(reqBody) )
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)

			//requestBody:=data.String()
			//stringLines:=strings.Split(requestBody,"\n")
			//for _,v:=range stringLines{
			//	obj:=map[string]interface{}{}
			//	err:=util.FromJSONBytes([]byte(v),&obj)
			//	if err!=nil{
			//		log.Error("invalid json,",util.SubString(v,0,512),err)
			//		break
			//	}
			//}
		}
		if joint.GetBool("warm_retry_message",false){
			log.Errorf("invalid bulk response, %v - %v",resp.StatusCode(),util.SubString(string(resbody),0,512))
		}
		return false
	}

	return true
}

var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}
