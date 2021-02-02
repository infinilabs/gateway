package indexing

import (
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
	bulkSizeInByte:= 1048576 * bulkSizeInMB
	if bulkSizeInKB>0{
		bulkSizeInByte= 1024 * bulkSizeInKB
	}

	meta := elastic.GetMetadata(elasticsearch)
	wg := sync.WaitGroup{}

	if meta==nil{
		return errors.New("metadata is nil")
	}

	totalSize := 0
	esInstanceVal := joint.MustGetString("elasticsearch")
	indices, isIndex := joint.GetStringArray("index")

	//index,shard,level
	if isIndex {
		for _, v := range indices {
			indexSettings := meta.Indices[v]
			for i := 0; i < indexSettings.Shards; i++ {
				queueName := common.GetShardLevelShuffleKey(esInstanceVal, v, i)
				shardInfo := meta.GetPrimaryShardInfo(v, i)
				nodeInfo := meta.GetNodeInfo(shardInfo.NodeID)
				for i := 0; i < workers; i++ {
					wg.Add(1)
					go joint.NewBulkWorker(&totalSize, bulkSizeInByte, &wg, queueName, nodeInfo.Http.PublishAddress)
				}
			}
		}
	} else { //node level
		if meta.Nodes==nil{
			return errors.New("nodes is nil")
		}
		for k, v := range meta.Nodes {
			queueName := common.GetNodeLevelShuffleKey(esInstanceVal, k)
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go joint.NewBulkWorker(&totalSize, bulkSizeInByte, &wg, queueName, v.Http.PublishAddress)
			}
		}
	}

	wg.Wait()

	return nil
}

func (joint BulkIndexingJoint) NewBulkWorker(count *int, bulkSizeInByte int, wg *sync.WaitGroup, queueName string, endpoint string) {
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

	log.Debug("start worker:", queueName)

	mainBuf := bytes.Buffer{}
	esInstanceVal := joint.MustGetString("elasticsearch")
	timeOut := joint.GetIntOrDefault("idle_timeout_in_second", 5)
	idleDuration := time.Duration(timeOut) * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()
	cfg := elastic.GetConfig(esInstanceVal)

READ_DOCS:
	for {

		idleTimeout.Reset(idleDuration)

		select {

		//each message is complete bulk message, must be end with \n
		case pop := <-queue.ReadChan(queueName):
			stats.IncrementBy("bulk", "bytes_received", int64(mainBuf.Len()))
			mainBuf.Write(pop)
			(*count)++
			if mainBuf.Len() > (bulkSizeInByte) {
				if global.Env().IsDebug {
					log.Trace("hit buffer size, ", mainBuf.Len())
				}
				goto CLEAN_BUFFER
			}

		case <-idleTimeout.C:
			if global.Env().IsDebug{
				log.Tracef("%v no message input", idleDuration)
			}
			goto CLEAN_BUFFER
		}

		goto READ_DOCS

	CLEAN_BUFFER:
		if mainBuf.Len() > 0 {
			success:=joint.Bulk(&cfg, endpoint, &mainBuf)
			log.Trace("clean buffer, and execute bulk insert")

			if !success{
				queue.Push(queueName,mainBuf.Bytes())
			}else{
				stats.IncrementBy("bulk", "bytes_processed", int64(mainBuf.Len()))
			}

			mainBuf.Reset()
			//TODO handle retry and fallback/over, dead letter queue
			//set services to failure, need manual restart
			//process dead letter queue first next round

		}

	}
}

func (joint BulkIndexingJoint) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data *bytes.Buffer) bool{
	if data == nil || data.Len() == 0 {
		return true
	}
	data.WriteRune('\n')

	if cfg.IsTLS() {
		endpoint = "https://" + endpoint
	} else {
		endpoint = "http://" + endpoint
	}
	url := fmt.Sprintf("%s/_bulk", endpoint)
	req := util.NewPostRequest(url, data.Bytes())

	req.SetContentType(util.ContentTypeJson)

	if cfg.BasicAuth != nil {
		req.SetBasicAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password)
	}

	if cfg.HttpProxy != "" {
		req.SetProxy(cfg.HttpProxy)
	}

	compress:=false
	_, err := joint.DoRequest(compress, http.MethodPost, url, cfg.BasicAuth.Username, cfg.BasicAuth.Password, data.Bytes(), "")

	//TODO handle error, retry and send to deadlock queue

	if err != nil {
		log.Error(err)
		path1:=path.Join(global.Env().GetWorkingDir(),"bulk_failure.log")
		util.FileAppendNewLineWithByte(path1,[]byte(url))
		util.FileAppendNewLineWithByte(path1,data.Bytes())
		util.FileAppendNewLineWithByte(path1,[]byte("error:\n"))
		util.FileAppendNewLineWithByte(path1,[]byte(err.Error()))
		return false
	}
	return true
}

var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func  (joint BulkIndexingJoint)DoRequest(compress bool, method string, loadUrl string, user, password string, body []byte, proxy string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	req.Reset()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(loadUrl)
	req.Header.SetMethod(method)
	req.Header.SetUserAgent("bulk_indexing")

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	req.Header.SetContentType("application/json")

	if user != "nil" {
		req.URI().SetUsername(user)
		req.URI().SetPassword(password)
	}

	if len(body) > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), body, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			req.SetBody(body)

		}
	}
	retryTimes:=0

	DO:

	err := fastHttpClient.Do(req, resp)

	if err != nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return nil, err
	}

	if resp == nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return nil, err
	}


	// Do we need to decompress the response?
	var resbody =resp.GetRawBody()
	if global.Env().IsDebug{
		log.Trace(resp.StatusCode(),string(resbody))
	}

	if resp.StatusCode()==400{
		path1:=path.Join(global.Env().GetWorkingDir(),"bulk_400_failure.log")
		util.FileAppendNewLineWithByte(path1,[]byte(loadUrl))
		util.FileAppendNewLineWithByte(path1,body)
		util.FileAppendNewLineWithByte(path1,resbody)
		return nil, errors.New("400 error")
	}

	//TODO check respbody's error
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		//200{"took":2,"errors":true,"items":[
		if resp.StatusCode()==http.StatusOK{
			//handle error items
			//"errors":true
			hit:=util.LimitedBytesSearch(resbody,[]byte("\"errors\":true"),64)
			if hit{
				log.Warnf("elasticsearch bulk error, retried %v times, will try again",retryTimes)
				retryTimes++
				delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
				time.Sleep(time.Duration(delayTime)*time.Second)
				goto DO
			}
		}

		return resbody, nil
	} else if resp.StatusCode()==429 {
		log.Warnf("elasticsearch rejected, retried %v times, will try again",retryTimes)
		delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
		time.Sleep(time.Duration(delayTime)*time.Second)
		if retryTimes>300{
			log.Errorf("elasticsearch rejected, retried %v times, quit retry",retryTimes)
			return resbody,errors.New("elasticsearch rejected")
		}
		retryTimes++
		goto DO
	}else {
		return resbody,errors.Errorf("invalid bulk response, %v",string(resbody))
	}

	return nil, nil
}
