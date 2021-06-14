package queue_consumer

import (
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
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
	param.Parameters
	inputQueueName string
}

func (joint DiskQueueConsumer) Name() string {
	return "disk_queue_consumer"
}

var fastHttpClient = &fasthttp.Client{
	Name: "disk_queue_consumer",
	DisableHeaderNamesNormalizing: false,
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func (joint DiskQueueConsumer) Process(c *pipeline.Context) error {
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

	workers, _ := joint.GetInt("worker_size", 1)
	joint.inputQueueName = joint.MustGetString("input_queue")

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go joint.NewBulkWorker(&totalSize, &wg)
	}

	wg.Wait()

	return nil
}

func (joint DiskQueueConsumer) NewBulkWorker(count *int, wg *sync.WaitGroup) {

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

	timeOut := joint.GetIntOrDefault("idle_timeout_in_second", 5)
	esInstanceVal := joint.MustGetString("elasticsearch")
	waitingAfter,waitingOtherQueue := joint.GetStringArray("waiting_after")
	esConfig := elastic.GetConfig(esInstanceVal)

	idleDuration := time.Duration(timeOut) * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()

	onErrorQueue:=joint.inputQueueName+"_pending"
	onDeadLetterQueue:=joint.inputQueueName+"_dead_letter"

READ_DOCS:
	for {
		idleTimeout.Reset(idleDuration)

		if waitingOtherQueue{
			if len(waitingAfter)>0{
				for _,v:=range waitingAfter{
					depth:=queue.Depth(v)
					if depth>0{
						log.Debugf("%v has pending %v messages, cleanup it first",v,depth)
						time.Sleep(5*time.Second)
						goto READ_DOCS
					}
				}
			}
		}

		//handle message on error queue
		if queue.Depth(onErrorQueue)>0{
			log.Debug(onErrorQueue," has pending message, clear it first")
			goto HANDLE_PENDING
		}

		select {

		case pop := <-queue.ReadChan(joint.inputQueueName):
			ok,status,err:=processMessage(esConfig,pop)
			if !ok{
				if global.Env().IsDebug{
					log.Debug(ok,status,err)
				}
				if status>=400 && status< 500{
					err:=queue.Push(onDeadLetterQueue,pop)
					if err!=nil{
						panic(err)
					}
				}else{
					err:=queue.Push(onErrorQueue,pop)
					if err!=nil{
						panic(err)
					}
				}
				time.Sleep(5*time.Second)
			}
		case <-idleTimeout.C:
			if global.Env().IsDebug{
				log.Tracef("%v no message input", idleDuration)
			}
		}

		goto READ_DOCS

	}

HANDLE_PENDING:
	idleTimeout1 := time.NewTimer(idleDuration)
	defer idleTimeout1.Stop()

	log.Trace("handle pending messages ",onErrorQueue)

	for {
		idleTimeout1.Reset(idleDuration)
		select {

		case pop := <-queue.ReadChan(onErrorQueue):

			ok,status,err:=processMessage(esConfig,pop)
			if !ok{
				if global.Env().IsDebug{
					log.Debug(ok,status,err)
				}
				if status>401 && status< 500{
					log.Error("push to dead letter queue,",onDeadLetterQueue)
					err:=queue.Push(onDeadLetterQueue,pop)
					if err!=nil{
						panic(err)
					}
				}else{
					err:=queue.Push(onErrorQueue,pop)
					if err!=nil{
						panic(err)
					}
				}

				time.Sleep(1*time.Second)
			}
		case <-idleTimeout1.C:
			if global.Env().IsDebug{
				log.Tracef("%v no message input", idleDuration)
			}
			goto READ_DOCS
		}
	}
}

func processMessage(esConfig *elastic.ElasticsearchConfig,pop []byte)(bool,int,error)  {
	req:=fasthttp.AcquireRequest()
	err:=req.Decode(pop)
	if err!=nil{
		return false,0,err
	}

	endpoint:=esConfig.GetHost()

	req.Header.SetHost(endpoint)
	resp:=fasthttp.AcquireResponse()
	err=fastHttpClient.Do(req, resp)
	if esConfig.TrafficControl!=nil{
	RetryRateLimit:

		if esConfig.TrafficControl.MaxQpsPerNode>0{
			if !rate.GetRateLimiterPerSecond(esConfig.Name,endpoint+"max_qps", int(esConfig.TrafficControl.MaxQpsPerNode)).Allow(){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}

		if esConfig.TrafficControl.MaxBytesPerNode>0{
			if !rate.GetRateLimiterPerSecond(esConfig.Name,endpoint+"max_bps", int(esConfig.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(),req.GetRequestLength()){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}

	}


	if err != nil {
		log.Error(err)
		return false,resp.StatusCode(), err
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated || resp.StatusCode() == http.StatusNotFound {
		if resp.StatusCode() == http.StatusOK && util.ContainStr(string(req.RequestURI()),"_bulk") {
			//handle bulk response partial failure
			data:=map[string]interface{}{}
			util.FromJSONBytes(resp.GetRawBody(),&data)
			err,ok2:=data["errors"]
			if ok2{
				if err==true{
					if global.Env().IsDebug{
						log.Error("disk_queue checking bulk response, invalid, ",ok2,",",err,",",util.SubString(string(resp.GetRawBody()),0,256))
					}
					return false,resp.StatusCode(), errors.New(fmt.Sprintf("%v",err))
				}
			}
		}
		return true,resp.StatusCode(),nil
	}else{
		return false,resp.StatusCode(), errors.New(fmt.Sprintf("invalid status code, %v %v %v",resp.StatusCode(),err,util.SubString(string(resp.GetRawBody()),0,1024)))
	}

}