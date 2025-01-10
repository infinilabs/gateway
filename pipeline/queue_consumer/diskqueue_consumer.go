// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package queue_consumer

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

var defaultHTTPPool = fasthttp.NewRequestResponsePool("queue_consumer")

type DiskQueueConsumer struct {
	config Config
}

func (processor *DiskQueueConsumer) Name() string {
	return "queue_consumer"
}

type Config struct {
	NumOfWorkers        int `config:"worker_size"`
	IdleTimeoutInSecond int `config:"idle_timeout_in_seconds"`

	FetchMinBytes    int `config:"fetch_min_bytes"`
	FetchMaxBytes    int `config:"fetch_max_bytes"`
	FetchMaxMessages int `config:"fetch_max_messages"`
	FetchMaxWaitMs   int `config:"fetch_max_wait_ms"`

	SaveSuccessDocsToQueue bool   `config:"save_partial_success_requests"`
	PartialSuccessQueue    string `config:"partial_success_queue"`

	InputQueue   string `config:"input_queue"`
	FailureQueue string `config:"failure_queue"`
	InvalidQueue string `config:"invalid_queue"`

	Elasticsearch string   `config:"elasticsearch"`
	WaitingAfter  []string `config:"waiting_after"`
	Compress      bool     `config:"compress"`

	SafetyParse   bool `config:"safety_parse"`
	DocBufferSize int  `config:"doc_buffer_size"`
}

func init() {
	pipeline.RegisterProcessorPlugin("queue_consumer", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers:        1,
		IdleTimeoutInSecond: 5,
		DocBufferSize:       256 * 1024,
		SafetyParse:         true,
		FetchMinBytes:       1,
		FetchMaxMessages:    10,
		FetchMaxWaitMs:      10000,
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
	MaxConnsPerHost:               1000,
	Name:                          "queue_consumer",
	DisableHeaderNamesNormalizing: false,
	TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
	DialDualStack:                 true,
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
				ctx.RecordError(fmt.Errorf("queue consumer panic: %v", r))
			}
		}
		wg.Done()
	}()

	esInstanceVal := processor.config.Elasticsearch
	waitingAfter := processor.config.WaitingAfter
	metadata := elastic.GetMetadata(esInstanceVal)
	if metadata == nil {
		panic(errors.Errorf("cluster metadata [%v] not ready", processor.config.Elasticsearch))
	}

	//idleDuration := time.Duration(timeOut) * time.Second
	if processor.config.FailureQueue == "" {
		processor.config.FailureQueue = processor.config.InputQueue + "-failure"
	}
	if processor.config.InvalidQueue == "" {
		processor.config.InvalidQueue = processor.config.InputQueue + "-invalid"
	}

	if processor.config.PartialSuccessQueue == "" {
		processor.config.PartialSuccessQueue = processor.config.InputQueue + "-partial"
	}

	qConfig := queue.GetOrInitConfig(processor.config.InputQueue)
	var consumer = queue.GetOrInitConsumerConfig(qConfig.ID, "group-001", "consumer-001")
	var initOfffset queue.Offset
	var offset queue.Offset

	//acquire consumer
	consumerInstance, err := queue.AcquireConsumer(qConfig, consumer, ctx.ID())
	defer queue.ReleaseConsumer(qConfig, consumer, consumerInstance)

	if err != nil || consumerInstance == nil {
		panic(err)
	}

	ctx1 := &queue.Context{}

READ_DOCS:
	initOfffset, _ = queue.GetOffset(qConfig, consumer)
	offset = initOfffset

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
				qCfg := queue.GetOrInitConfig(v)
				//TODO handle consumer type
				hasLag := queue.HasLag(qCfg)
				if hasLag {
					log.Debugf("%v has pending messages, cleanup it first", v)
					time.Sleep(5 * time.Second)
					goto READ_DOCS
				}
			}
		}

		consumer.KeepActive()
		messages, _, err := consumerInstance.FetchMessages(ctx1, consumer.FetchMaxMessages)

		if len(messages) == 0 {
			time.Sleep(time.Millisecond * time.Duration(500))
		}

		if len(messages) > 0 {
			for _, pop := range messages {
				if err != nil {
					log.Error(err)
					panic(err)
				}

				if len(pop.Data) > 0 {
					ok, status, err := processor.processMessage(metadata, &pop)
					if err != nil {
						log.Error(ok, status, err)
					}

					if !ok {
						if global.Env().IsDebug {
							log.Debug(ok, status, err)
						}

						if status != 429 && status >= 400 && status < 500 {
							log.Error("push to dead letter queue:", processor.config.InvalidQueue, ",", err)
							err := queue.Push(queue.GetOrInitConfig(processor.config.InvalidQueue), pop.Data)
							if err != nil {
								panic(err)
							}
						} else {
							err := queue.Push(queue.GetOrInitConfig(processor.config.FailureQueue), pop.Data)
							if err != nil {
								panic(err)
							}
						}
					}
				}
				offset = pop.NextOffset
			}

			if !offset.Equals(initOfffset) {
				ok, err := queue.CommitOffset(qConfig, consumer, offset)
				if !ok || err != nil {
					panic(err)
				}
				initOfffset = offset
			}
		}

	}
}

func gzipBest(a *[]byte) []byte {
	var b bytes.Buffer
	gz, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
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

func (processor *DiskQueueConsumer) processMessage(metadata *elastic.ElasticsearchMetadata, msg *queue.Message) (bool, int, error) {
	req := defaultHTTPPool.AcquireRequestWithTag("disk_consumer_request")
	defer defaultHTTPPool.ReleaseRequest(req)
	err := req.Decode(msg.Data)
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
	orignalSchema := string(req.PhantomURI().Scheme())
	orignalHost := string(req.PhantomURI().Host())
	clonedURI := req.CloneURI()
	defer fasthttp.ReleaseURI(clonedURI)
	if metadata.GetSchema() != orignalSchema {
		clonedURI.SetScheme(metadata.GetSchema())
		req.SetURI(clonedURI)
	}

	host := metadata.GetActiveHost()
	req.SetHost(host)
	resp := defaultHTTPPool.AcquireResponseWithTag("disk_consumer_response")
	defer defaultHTTPPool.ReleaseResponse(resp)

	acceptGzipped := req.AcceptGzippedResponse()
	compressed := false
	if !req.IsGzipped() && processor.config.Compress {
		data := req.Body()
		data1 := gzipBest(&data)

		//TODO handle response, if client not support gzip, return raw body
		req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
		req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
		req.SwapBody(data1)
		compressed = true
	}

	metadata.CheckNodeTrafficThrottle(util.UnsafeBytesToString(req.Header.Host()), 1, req.GetRequestLength(), 0)

	//execute
	err = fastHttpClient.Do(req, resp)

	if !acceptGzipped && compressed {
		body := resp.GetRawBody()
		resp.SwapBody(body)
		resp.Header.Del(fasthttp.HeaderContentEncoding)
		resp.Header.Del(fasthttp.HeaderContentEncoding2)
	}

	// restore schema
	clonedURI.SetScheme(orignalSchema)
	req.SetURI(clonedURI)
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
		if util.ContainStr(string(req.Header.RequestURI()), "_bulk") {

			//var resbody = resp.GetRawBody()
			//requestBytes := req.GetRawBody()
			//
			//nonRetryableItems := bytebufferpool.Get()
			//retryableItems := bytebufferpool.Get()
			//successItems := bytebufferpool.Get()
			//
			////TODO remove
			//containError:=processor.HandleBulkResponse(processor.config.SafetyParse,requestBytes,resbody,processor.config.DocBufferSize,nonRetryableItems,retryableItems,successItems)
			//if containError {
			//
			//	log.Error("error in bulk requests,", resp.StatusCode(), util.SubString(string(resbody), 0, 256))
			//
			//	if nonRetryableItems.Len() > 0 {
			//		nonRetryableItems.WriteByte('\n')
			//		bytes := req.OverrideBodyEncode(nonRetryableItems.Bytes(), true)
			//		queue.Push(queue.GetOrInitConfig(processor.config.InvalidQueue), bytes)
			//		bytebufferpool.Put(nonRetryableItems)
			//	}
			//
			//	if retryableItems.Len() > 0 {
			//		retryableItems.WriteByte('\n')
			//		bytes := req.OverrideBodyEncode(retryableItems.Bytes(), true)
			//		queue.Push(queue.GetOrInitConfig(processor.config.FailureQueue), bytes)
			//		bytebufferpool.Put(retryableItems)
			//	}
			//
			//
			//	if successItems.Len() > 0 && processor.config.SaveSuccessDocsToQueue {
			//		successItems.WriteByte('\n')
			//		bytes := req.OverrideBodyEncode(successItems.Bytes(), true)
			//		queue.Push(queue.GetOrInitConfig(processor.config.PartialSuccessQueue), bytes)
			//		bytebufferpool.Put(successItems)
			//	}
			//
			//	//save message bytes, with metadata, set codec to wrapped bulk messages
			//	queue.Push(queue.GetOrInitConfig("failure_messages"), util.MustToJSONBytes(util.MapStr{
			//		"offset":msg.Offset,
			//		"response":string(resbody),
			//	}))
			//
			//
			//}
		}
		return true, resp.StatusCode(), nil
	} else {
		if global.Env().IsDebug {
			log.Warn(err, resp.StatusCode(), req.Header.String(), util.SubString(util.UnsafeBytesToString(req.GetRawBody()), 0, 512), util.SubString(util.UnsafeBytesToString(respBody), 0, 256))
		}
		return false, resp.StatusCode(), errors.New(fmt.Sprintf("invalid status code, %v %v %v", resp.StatusCode(), err, util.SubString(util.UnsafeBytesToString(respBody), 0, 256)))
	}

}
