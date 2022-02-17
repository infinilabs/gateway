package queue_consumer

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	es "infini.sh/gateway/proxy/filters/elastic"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type DiskQueueConsumer struct {
	config Config
}

func (processor *DiskQueueConsumer) Name() string {
	return "queue_consumer"
}

type Config struct {
	NumOfWorkers        int      `config:"worker_size"`
	IdleTimeoutInSecond int      `config:"idle_timeout_in_seconds"`
	InputQueue          string   `config:"input_queue"`

	FailureQueue        string   `config:"failure_queue"`
	InvalidQueue        string   `config:"invalid_queue"`

	Elasticsearch       string   `config:"elasticsearch"`
	WaitingAfter        []string `config:"waiting_after"`
	Compress            bool     `config:"compress"`

	SafetyParse bool `config:"safety_parse"`
	DocBufferSize int `config:"doc_buffer_size"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers:        1,
		IdleTimeoutInSecond: 5,
		DocBufferSize: 256*1024,
		SafetyParse: true,
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of urldecode processor: %s", err)
	}

	processor := &DiskQueueConsumer{
		config: cfg,
	}

	return processor, nil
}

var fastHttpClient = &fasthttp.Client{
	MaxConnsPerHost: 1000,
	Name:                          "queue_consumer",
	DisableHeaderNamesNormalizing: false,
	TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
}

func (processor *DiskQueueConsumer) Process(ctx *pipeline.Context) error {
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
				log.Error("error in queue_consumer,", v)
			}
		}
	}()

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < processor.config.NumOfWorkers; i++ {
		wg.Add(1)
		go processor.NewBulkWorker(ctx, &totalSize, &wg)
	}

	wg.Wait()
	return nil
}

func (processor *DiskQueueConsumer) NewBulkWorker(ctx *pipeline.Context, count *int, wg *sync.WaitGroup) {

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
				log.Error("error in json indexing worker,", v)
				ctx.Failed()
			}
		}
		wg.Done()
	}()

	timeOut := processor.config.IdleTimeoutInSecond
	esInstanceVal := processor.config.Elasticsearch
	waitingAfter := processor.config.WaitingAfter
	metadata := elastic.GetMetadata(esInstanceVal)
	if metadata==nil{
		panic(errors.Errorf("cluster metadata [%v] not ready", processor.config.Elasticsearch))
	}


	idleDuration := time.Duration(timeOut) * time.Second
	if processor.config.FailureQueue == "" {
		processor.config.FailureQueue = processor.config.InputQueue + "-failure"
	}
	if processor.config.InvalidQueue == "" {
		processor.config.InvalidQueue = processor.config.InputQueue + "-invalid"
	}


READ_DOCS:
	for {

		if ctx.IsCanceled() {
			//log.Error("received cancel signal:")
			return
		}

		if !metadata.IsAvailable() {
			log.Debugf("cluster [%v] is not available, task stop", metadata.Config.Name)
			return
		}

		if len(waitingAfter) > 0 {
			for _, v := range waitingAfter {
				qCfg:=queue.GetOrInitConfig(v)
				depth := queue.Depth(qCfg)
				if depth > 0 {
					log.Debugf("%v has pending %v messages, cleanup it first", v, depth)
					time.Sleep(5 * time.Second)
					goto READ_DOCS
				}
			}
		}

		pop, _, err := queue.PopTimeout(queue.GetOrInitConfig(processor.config.InputQueue), idleDuration)
		if err != nil {
			log.Error(err)
			panic(err)
		}

		if len(pop) > 0 {
			ok, status, err := processor.processMessage(metadata, pop)
			if err != nil {
				log.Error(ok, status, err)
			}

			if !ok {
				if global.Env().IsDebug {
					log.Debug(ok, status, err)
				}

				if status != 429 && status >= 400 && status < 500 {
					log.Error("push to dead letter queue:", processor.config.InvalidQueue, ",", err)
					err := queue.Push(queue.GetOrInitConfig(processor.config.InvalidQueue), pop)
					if err != nil {
						panic(err)
					}
				} else {
					err := queue.Push(queue.GetOrInitConfig(processor.config.FailureQueue), pop)
					if err != nil {
						panic(err)
					}
				}
			}
		}

	}
}

func gzipBest(a *[]byte) []byte {
	var b bytes.Buffer
	gz,err := gzip.NewWriterLevel(&b,gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	if _, err := gz.Write(*a); err != nil {
		gz.Close()
		panic(err)
	}
	gz.Close()
	return b.Bytes()
}

func (processor *DiskQueueConsumer) processMessage(metadata *elastic.ElasticsearchMetadata, pop []byte) (bool, int, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	err := req.Decode(pop)
	if err != nil {
		log.Error("failed to decode request, ", metadata.Config.Name)
		return false, 408, err
	}

	if global.Env().IsDebug {
		log.Trace(err)
		log.Trace(req.Header.String())
		log.Trace(string(req.GetRawBody()))
	}

	// modify schema，align with elasticsearch's schema
	orignalSchema := string(req.URI().Scheme())
	orignalHost := string(req.URI().Host())
	if metadata.GetSchema() != orignalSchema {
		req.URI().SetScheme(metadata.GetSchema())
	}

	host := metadata.GetActiveHost()
	req.SetHost(host)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	acceptGzipped:=req.AcceptGzippedResponse()
	compressed:=false
	if !req.IsGzipped() && processor.config.Compress {
		data := req.Body()
		data1:=gzipBest(&data)

		//TODO handle response, if client not support gzip, return raw body
		req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
		req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
		req.SwapBody(data1)
		compressed=true
	}

	metadata.CheckNodeTrafficThrottle(util.UnsafeBytesToString(req.Header.Host()),1,req.GetRequestLength(),0)

	//execute
	err = fastHttpClient.Do(req, resp)

	if !acceptGzipped&&compressed{
		body:=resp.GetRawBody()
		resp.SwapBody(body)
		resp.Header.Del(fasthttp.HeaderContentEncoding)
		resp.Header.Del(fasthttp.HeaderContentEncoding2)
	}

	// restore schema
	req.URI().SetScheme(orignalSchema)
	req.SetHost(orignalHost)

	if err != nil {
		return false, resp.StatusCode(), err
	}

	if global.Env().IsDebug {
		log.Trace(err)
		log.Trace(resp.StatusCode())
		log.Trace(string(resp.GetRawBody()))
	}

	respBody := resp.GetRawBody()

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated || resp.StatusCode() == http.StatusNotFound {
		if util.ContainStr(string(req.RequestURI()), "_bulk") {

			var resbody = resp.GetRawBody()
			requestBytes := req.GetRawBody()

			nonRetryableItems := bytebufferpool.Get()
			retryableItems := bytebufferpool.Get()

			containError:=es.HandleBulkResponse(processor.config.SafetyParse,requestBytes,resbody,processor.config.DocBufferSize,nonRetryableItems,retryableItems)
			if containError {

				log.Error("error in bulk requests,", resp.StatusCode(), util.SubString(string(resbody), 0, 256))

				if nonRetryableItems.Len() > 0 {
					nonRetryableItems.WriteByte('\n')
					bytes := req.OverrideBodyEncode(nonRetryableItems.Bytes(), true)
					queue.Push(queue.GetOrInitConfig(processor.config.InvalidQueue), bytes)
					bytebufferpool.Put(nonRetryableItems)
				}

				if retryableItems.Len() > 0 {
					retryableItems.WriteByte('\n')
					bytes := req.OverrideBodyEncode(retryableItems.Bytes(), true)
					queue.Push(queue.GetOrInitConfig(processor.config.FailureQueue), bytes)
					bytebufferpool.Put(retryableItems)
				}

				//save message bytes, with metadata, set codec to wrapped bulk messages
				queue.Push(queue.GetOrInitConfig("failure_messages"), util.MustToJSONBytes(buffer.GetMessageStatus(true)))


			}
		}
		return true, resp.StatusCode(), nil
	} else {
		if global.Env().IsDebug {
			log.Warn(err, resp.StatusCode(),req.Header.String(), util.SubString(string(req.GetRawBody()), 0, 512), util.SubString(string(respBody), 0, 256))
		}
		return false, resp.StatusCode(), errors.New(fmt.Sprintf("invalid status code, %v %v %v", resp.StatusCode(), err, util.SubString(string(respBody), 0, 256)))
	}

}
