package queue_consumer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
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
	//log.Error("DiskQueueConsumer finished.")
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
				ctx.Failed()
			}
		}
		wg.Done()
		//log.Error("DiskQueueConsumer inner finished.")
	}()

	timeOut := processor.config.IdleTimeoutInSecond
	esInstanceVal := processor.config.Elasticsearch
	waitingAfter := processor.config.WaitingAfter
	metadata := elastic.GetMetadata(esInstanceVal)

	idleDuration := time.Duration(timeOut) * time.Second
	pendingQueue := processor.config.InputQueue + "_pending"
	deadLetterQueue := processor.config.InputQueue + "_dead_letter"

READ_DOCS:
	for {

		if ctx.IsCanceled(){
			//log.Error("received cancel signal:")
			return
		}

		if !metadata.IsAvailable(){
			log.Debugf("cluster [%v] is not available, task stop",metadata.Config.Name)
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
		if queue.Depth(pendingQueue) > 0 {
			log.Debug(pendingQueue, " has pending message, clear it first")
			goto HANDLE_PENDING
		}

		pop, _, err := queue.PopTimeout(processor.config.InputQueue, idleDuration)
		if err!=nil{
			log.Error(err)
			panic(err)
		}

		if len(pop)>0{
			ok, status, err := processMessage(metadata, pop)
			if err!=nil{
				log.Error(err)
			}

			if !ok {
				if global.Env().IsDebug {
					log.Debug(ok, status, err)
				}

				//TODO handle 429

				if status >= 400 && status < 500 {
					log.Error("push to dead letter queue:", deadLetterQueue, ",", err)
					err := queue.Push(deadLetterQueue, pop)
					if err != nil {
						panic(err)
					}
				} else {
					err := queue.Push(pendingQueue, pop)
					if err != nil {
						panic(err)
					}
				}
			}
		}



	}

HANDLE_PENDING:

	log.Trace("handle pending messages ", pendingQueue)
	for {

		if ctx.IsCanceled(){
			//log.Error("received cancel signal:")
			return
		}

		if !metadata.IsAvailable(){
			log.Errorf("cluster [%v] is not available, task stop",metadata.Config.Name)
			return
		}

		pop, timeout, err := queue.PopTimeout(pendingQueue, idleDuration)

		//if no pending message left, goto the main progress
		if timeout{
			goto READ_DOCS
		}

		if len(pop)>0{
			ok, status, err := processMessage(metadata, pop)
			if err!=nil{
				log.Error(err)
			}

			if !ok {
				if global.Env().IsDebug {
					log.Debug(ok, status, err)
				}

				if status > 401 && status < 500 {
					log.Error("push to dead letter queue:", deadLetterQueue, ",", err)
					err := queue.Push(deadLetterQueue, pop)
					if err != nil {
						panic(err)
					}
				} else {
					err := queue.Push(pendingQueue, pop)
					if err != nil {
						panic(err)
					}
				}
			}
		}
		if err!=nil{
			log.Error(err)
			panic(err)
		}
	}
}

func processMessage(metadata *elastic.ElasticsearchMetadata, pop []byte) (bool, int, error) {
	req := fasthttp.AcquireRequest()
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
	orignalSchema:=string(req.URI().Scheme())
	orignalHost:=string(req.URI().Host())
	if metadata.GetSchema()!=orignalSchema{
		req.URI().SetScheme(metadata.GetSchema())
	}

	host := metadata.GetActiveHost()
	req.SetHost(host)
	resp := fasthttp.AcquireResponse()
	err = fastHttpClient.Do(req, resp)

	// restore schema
	req.URI().SetScheme(orignalSchema)
	req.SetHost(orignalHost)

	if err != nil {
		return false, resp.StatusCode(), err
	}

	if metadata.Config.TrafficControl != nil {

		if metadata.Config.TrafficControl.MaxWaitTimeInMs<=0{
			metadata.Config.TrafficControl.MaxWaitTimeInMs=10*1000
		}
		maxTime:=time.Duration(metadata.Config.TrafficControl.MaxWaitTimeInMs)*time.Millisecond
		startTime:=time.Now()
	RetryRateLimit:

		if time.Now().Sub(startTime)<maxTime{
			if metadata.Config.TrafficControl.MaxQpsPerNode > 0 {
				if !rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"max_qps", int(metadata.Config.TrafficControl.MaxQpsPerNode)).Allow() {
					stats.Increment(metadata.Config.ID,host+"-max_qps_throttled")
					if global.Env().IsDebug {
						log.Tracef("throttle request [%v] to upstream [%v]", req.URI().String(), host)
					}
					time.Sleep(10 * time.Millisecond)
					goto RetryRateLimit
				}
			}

			if metadata.Config.TrafficControl.MaxBytesPerNode > 0 {
				if !rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"max_bps", int(metadata.Config.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(), req.GetRequestLength()) {
					stats.Increment(metadata.Config.ID,host+"-max_bps_throttled")
					if global.Env().IsDebug {
						log.Tracef("throttle request [%v] to upstream [%v]", req.URI().String(), host)
					}
					time.Sleep(10 * time.Millisecond)
					goto RetryRateLimit
				}
			}
		}else{
			log.Warn("reached max traffic control time, throttle quitting")
		}
	}

	if global.Env().IsDebug {
		log.Trace(err)
		log.Trace(resp.StatusCode())
		log.Trace(string(resp.GetRawBody()))
	}

	stats.Increment("diskqueue_consumer",util.IntToString(resp.StatusCode()))

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated || resp.StatusCode() == http.StatusNotFound {
		if util.ContainStr(string(req.RequestURI()), "_bulk") {
			//handle bulk response partial failure
			va,err:=jsonparser.GetBoolean(resp.GetRawBody(),"errors")
			if va{
				stats.Increment("diskqueue_consumer","bulk_requests_errors")
				log.Error("error in bulk requests,",util.SubString(string(resp.GetRawBody()), 0, 256))
				time.Sleep(1*time.Second)
				return false, resp.StatusCode(), errors.New(fmt.Sprintf("%v", err))
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
