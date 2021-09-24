package queue_consumer

import (
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/elastic/model"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
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
	Elasticsearch       string   `config:"elasticsearch"`
	WaitingAfter        []string `config:"waiting_after"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers:        1,
		IdleTimeoutInSecond: 5,
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
				log.Error("error in json indexing,", v)
			}
		}
	}()

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < processor.config.NumOfWorkers; i++ {
		wg.Add(1)
		go processor.NewBulkWorker(ctx,&totalSize, &wg)
	}

	wg.Wait()

	return nil
}

func (processor *DiskQueueConsumer) NewBulkWorker(ctx *pipeline.Context,count *int, wg *sync.WaitGroup) {

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
				wg.Done()
			}
		}
	}()

	timeOut := processor.config.IdleTimeoutInSecond
	esInstanceVal := processor.config.Elasticsearch
	waitingAfter := processor.config.WaitingAfter
	esConfig := elastic.GetConfig(esInstanceVal)

	idleDuration := time.Duration(timeOut) * time.Second
	onErrorQueue := processor.config.InputQueue + "_pending"
	onDeadLetterQueue := processor.config.InputQueue + "_dead_letter"

READ_DOCS:
	for {

		if ctx.IsCanceled(){
			return
		}

		if len(waitingAfter) > 0 {
			for _, v := range waitingAfter {
				depth := queue.Depth(v)
				if depth > 0 {
					log.Debugf("%v has pending %v messages, cleanup it first", v, depth)
					time.Sleep(5 * time.Second)
					goto READ_DOCS
				}
			}
		}

		//handle message on error queue
		if queue.Depth(onErrorQueue) > 0 {
			log.Debug(onErrorQueue, " has pending message, clear it first")
			goto HANDLE_PENDING
		}

		pop, _, err := queue.PopTimeout(processor.config.InputQueue, idleDuration)

		if len(pop)>0{
			ok, status, err := processMessage(esConfig, pop)
			if !ok {
				if global.Env().IsDebug {
					log.Debug(ok, status, err)
				}
				if status >= 400 && status < 500 {
					log.Error("push to dead letter queue:", onDeadLetterQueue, ",", err)
					err := queue.Push(onDeadLetterQueue, pop)
					if err != nil {
						panic(err)
					}
				} else {
					err := queue.Push(onErrorQueue, pop)
					if err != nil {
						panic(err)
					}
				}
				time.Sleep(5 * time.Second)
			}
		}

		if err!=nil{
			log.Error(err)
			panic(err)
		}

	}

HANDLE_PENDING:

	log.Trace("handle pending messages ", onErrorQueue)
	for {

		if ctx.IsCanceled(){
			return
		}

		pop, _, err := queue.PopTimeout(onErrorQueue, idleDuration)
		if len(pop)>0{
			ok, status, err := processMessage(esConfig, pop)
			if !ok {
				if global.Env().IsDebug {
					log.Debug(ok, status, err)
				}
				if status > 401 && status < 500 {
					log.Error("push to dead letter queue:", onDeadLetterQueue, ",", err)
					err := queue.Push(onDeadLetterQueue, pop)
					if err != nil {
						panic(err)
					}
				} else {
					err := queue.Push(onErrorQueue, pop)
					if err != nil {
						panic(err)
					}
				}

				time.Sleep(1 * time.Second)
			}
		}
		if err!=nil{
			log.Error(err)
			panic(err)
		}
	}
}

func processMessage(esConfig *model.ElasticsearchConfig, pop []byte) (bool, int, error) {
	req := fasthttp.AcquireRequest()
	err := req.Decode(pop)
	if err != nil {
		log.Error("failed to decode request, ", esConfig.Name)
		return false, 408, err
	}

	if global.Env().IsDebug {
		log.Trace(err)
		log.Trace(req.Header.String())
		log.Trace(string(req.GetRawBody()))
	}

	endpoint := esConfig.GetHost()

	req.Header.SetHost(endpoint)
	resp := fasthttp.AcquireResponse()
	err = fastHttpClient.Do(req, resp)
	if err != nil {
		return false, resp.StatusCode(), err
	}

	if esConfig.TrafficControl != nil {
	RetryRateLimit:

		if esConfig.TrafficControl.MaxQpsPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(esConfig.Name, endpoint+"max_qps", int(esConfig.TrafficControl.MaxQpsPerNode)).Allow() {
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}

		if esConfig.TrafficControl.MaxBytesPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(esConfig.Name, endpoint+"max_bps", int(esConfig.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(), req.GetRequestLength()) {
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}

	}

	if global.Env().IsDebug {
		log.Trace(err)
		log.Trace(resp.StatusCode())
		log.Trace(string(resp.GetRawBody()))
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated || resp.StatusCode() == http.StatusNotFound {
		if resp.StatusCode() == http.StatusOK && util.ContainStr(string(req.RequestURI()), "_bulk") {
			//handle bulk response partial failure
			data := map[string]interface{}{}
			util.FromJSONBytes(resp.GetRawBody(), &data)
			err, ok2 := data["errors"]
			if ok2 {
				if err == true {
					if global.Env().IsDebug {
						log.Error("checking bulk response, invalid, ", ok2, ",", err, ",", util.SubString(string(resp.GetRawBody()), 0, 256))
					}
					return false, resp.StatusCode(), errors.New(fmt.Sprintf("%v", err))
				}
			}
		}
		return true, resp.StatusCode(), nil
	} else {
		if global.Env().IsDebug {
			log.Warn(err, resp.StatusCode(), util.SubString(string(req.GetRawBody()), 0, 512), util.SubString(string(resp.GetRawBody()), 0, 256))
		}
		return false, resp.StatusCode(), errors.New(fmt.Sprintf("invalid status code, %v %v %v", resp.StatusCode(), err, util.SubString(string(resp.GetRawBody()), 0, 256)))
	}

}
