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
	bulkSizeInMB, _ := joint.GetInt("bulk_size_in_mb", 10)
	elasticsearch := joint.GetStringOrDefault("elasticsearch", "default")
	bulkSizeInMB = 1000000 * bulkSizeInMB

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
					go joint.NewBulkWorker(&totalSize, bulkSizeInMB, &wg, queueName, nodeInfo.Http.PublishAddress)
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
				go joint.NewBulkWorker(&totalSize, bulkSizeInMB, &wg, queueName, v.Http.PublishAddress)
			}
		}
	}

	wg.Wait()

	return nil
}

func (joint BulkIndexingJoint) NewBulkWorker(count *int, bulkSizeInMB int, wg *sync.WaitGroup, queueName string, endpoint string) {
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
	docBuf := bytes.Buffer{}

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
			stats.IncrementBy("bulk", "received", int64(mainBuf.Len()))
			docBuf.Write(pop)
			mainBuf.Write(docBuf.Bytes())

			docBuf.Reset()

			(*count)++

			if mainBuf.Len() > (bulkSizeInMB) {
				log.Trace("hit buffer size, ", mainBuf.Len())
				goto CLEAN_BUFFER
			}

		case <-idleTimeout.C:
			log.Tracef("%v no message input", idleDuration)
			goto CLEAN_BUFFER
		}

		goto READ_DOCS

	CLEAN_BUFFER:

		if docBuf.Len() > 0 {
			mainBuf.Write(docBuf.Bytes())
		}

		if mainBuf.Len() > 0 {
			success:=joint.Bulk(&cfg, endpoint, &mainBuf)

			if !success{
				queue.Push(queueName,mainBuf.Bytes())
			}
			//TODO handle retry and fallback/over, dead letter queue
			//set services to failure, need manual restart
			//process dead letter queue first next round

			stats.IncrementBy("bulk", "processed", int64(mainBuf.Len()))
			log.Trace("clean buffer, and execute bulk insert")
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

	//_, err := util.ExecuteRequest(req)

	_, err := joint.DoRequest(true, http.MethodPost, url, cfg.BasicAuth.Username, cfg.BasicAuth.Password, data.Bytes(), "")

	//TODO handle error, retry and send to deadlock queue

	if err != nil {
		log.Error(err)
		return false
	}

	data.Reset()
	return true
}

var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func  (joint BulkIndexingJoint)DoRequest(compress bool, method string, loadUrl string, user, password string, body []byte, proxy string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(loadUrl)
	req.Header.SetMethod(method)

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
	contentEncoding := resp.Header.Peek("Content-Encoding")
	var resbody []byte
	if bytes.Equal(contentEncoding, []byte("gzip")) {
		resbody, err = resp.BodyGunzip()
		if err!=nil{
			log.Error(err)
		}
	} else {
		resbody = resp.Body()
	}

	if global.Env().IsDebug{
		log.Trace(string(resbody))
	}

	//TODO check respbody's error
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {
		return resbody, nil
	} else if resp.StatusCode()==429 {
		log.Warnf("elasticsearch rejected, retried %v times, will try again",retryTimes)
		delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)

		time.Sleep(time.Duration(delayTime)*time.Second)

		retryTimes++
		if retryTimes>300{
			log.Errorf("elasticsearch rejected, retried %v times, quit retry",retryTimes)
			return resbody,errors.New("elasticsearch rejected")
		}
		goto DO
	}else {
		return resbody,errors.New("invalid bulk response")
	}

	return nil, nil
}
