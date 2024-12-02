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

package flow_runner

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type Config struct {
	FlowName                       string `config:"flow"`
	InputQueue                     string `config:"input_queue"`
	FlowMaxRunningTimeoutInSeconds int    `config:"flow_max_running_timeout_in_second"`
	CommitTimeoutInSeconds         int    `config:"commit_timeout_in_second"`

	SkipEmptyQueue bool `config:"skip_empty_queue"`

	CommitOnTag              string `config:"commit_on_tag"`
	IdleWaitTimeoutInSeconds int    `config:"idle_wait_timeout_in_second"`

	Consumer queue.ConsumerConfig `config:"consumer"`
}

var ctxPool = &sync.Pool{
	New: func() interface{} {
		c := fasthttp.RequestCtx{
			EnrichedMetadata: true,
		}
		return &c
	},
}

func acquireCtx() (ctx *fasthttp.RequestCtx) {
	x1 := ctxPool.Get().(*fasthttp.RequestCtx)
	x1.Reset()
	x1.Request.Reset()
	x1.Response.Reset()
	return x1
}

func releaseCtx(ctx *fasthttp.RequestCtx) {
	ctx.Reset()
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctxPool.Put(ctx)
}

type FlowRunnerProcessor struct {
	config *Config
}

var signalChannel = make(chan bool, 1)

func init() {
	pipeline.RegisterProcessorPlugin("flow_runner", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		Consumer: queue.ConsumerConfig{
			Group:                  "group-001",
			Name:                   "consumer-001",
			FetchMinBytes:          1,
			FetchMaxBytes:          20 * 1024 * 1024,
			FetchMaxMessages:       1000,
			EOFRetryDelayInMs:      500,
			FetchMaxWaitMs:         10000,
			ConsumeTimeoutInSeconds:         60,
			EOFMaxRetryTimes:         10,
			ClientExpiredInSeconds: 60,
		},
		SkipEmptyQueue:                 true,
		CommitOnTag:                    "",
		IdleWaitTimeoutInSeconds:       1,
		FlowMaxRunningTimeoutInSeconds: 60,
		CommitTimeoutInSeconds:         1,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := FlowRunnerProcessor{config: &cfg}
	return &runner, nil
}

func (processor FlowRunnerProcessor) Stop() error {
	signalChannel <- true
	return nil
}

func (processor *FlowRunnerProcessor) Name() string {
	return "flow_runner"
}

func (processor *FlowRunnerProcessor) Process(ctx *pipeline.Context) error {

	if global.Env().IsDebug{
		log.Debugf("start flow_runner [%v]", processor.config.FlowName)
		defer  log.Debugf("exit flow_runner [%v]", processor.config.FlowName)
	}

	var initOfffset queue.Offset
	var offset queue.Offset
	qConfig := queue.GetOrInitConfig(processor.config.InputQueue)
	var consumer = queue.GetOrInitConsumerConfig(qConfig.ID, processor.config.Consumer.Group, processor.config.Consumer.Name)
	initOfffset, _ = queue.GetOffset(qConfig, consumer)
	offset = initOfffset
	if processor.config.SkipEmptyQueue && !queue.ConsumerHasLag(qConfig, consumer) {
		log.Debug(processor.config.FlowName,", skip empty queue: ",qConfig.ID,",",qConfig.Name)
		time.Sleep(5*time.Second)
		return nil
	}

	flowProcessor := common.GetFlowProcess(processor.config.FlowName)
	var skipFinalDocsProcess bool

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

				if !util.ContainStr(v, "acquired by another node") {
					log.Errorf("error in flow_runner [%v], [%v]", processor.config.FlowName, v)
				}
				ctx.RecordError(fmt.Errorf("flow runner panic: %v", r))
				skipFinalDocsProcess = true
			}
		}

		if skipFinalDocsProcess {
			return
		}

		if !offset.Equals(initOfffset) {
			ok, err := queue.CommitOffset(qConfig, consumer, offset)
			log.Debugf("%v,%v commit offset:%v, result:%v,%v", qConfig.Name, consumer.Name, offset,ok,err)
			if !ok || err != nil {
				ctx.RecordError(fmt.Errorf("failed to commit offset, ok: %v, err: %v", ok, err))
			} else {
				initOfffset = offset
			}
		}
	}()

	t1 := util.AcquireTimer(time.Duration(processor.config.FlowMaxRunningTimeoutInSeconds) * time.Second)
	defer util.ReleaseTimer(t1)

	//acquire consumer
	consumerInstance, err := queue.AcquireConsumer(qConfig, consumer,ctx.ID())
	defer queue.ReleaseConsumer(qConfig, consumer,consumerInstance)

	if err != nil || consumerInstance == nil {
		panic(err)
	}

	ctx1 := &queue.Context{}
	ctx1.InitOffset=initOfffset

	lastCommitTime := time.Now()
	var commitIdle = time.Duration(processor.config.CommitTimeoutInSeconds) * time.Second
	for {
		if ctx.IsCanceled() {
			return nil
		}

		if global.ShuttingDown() {
			return nil
		}

		select {
		case <-t1.C:
			return nil
		case <-ctx.Context.Done():
			return nil
		case <-signalChannel:
			return nil
		default:
			if global.Env().IsDebug {
				log.Debug(qConfig.Name, ",", consumer.Group, ",", consumer.Name, ",init offset:", offset)
			}

			log.Debugf("star to consume queue:%v, %v", qConfig.Name,ctx1)
			processor.config.Consumer.KeepActive()
			messages, timeout, err :=consumerInstance.FetchMessages(ctx1, processor.config.Consumer.FetchMaxMessages)
			log.Debugf("get %v messages from queue:%v, %v", len(messages), qConfig.Name,ctx1)

			if err != nil && err.Error() != "EOF" {
				log.Error(err)
				panic(err)
			}

			start1:=time.Now()
			if len(messages) > 0 {
				for _, pop := range messages {
					ctx := acquireCtx()
					err = ctx.Request.Decode(pop.Data)
					if err != nil {
						log.Error(err)
						panic(err)
					}

					if global.Env().IsDebug {
						log.Tracef("start forward request to flow:%v", processor.config.FlowName)
					}

					ctx.SetFlowID(processor.config.FlowName)

					flowProcessor(ctx)

					if global.Env().IsDebug {
						log.Tracef("end forward request to flow:%v", processor.config.FlowName)
					}


					if processor.config.CommitOnTag != "" {
						tags, ok := ctx.GetTags()
						if ok {
							_, ok = tags[processor.config.CommitOnTag]
						}

						//log.Error("hit commit tag:",i,"=>",ok,",",pop.NextOffset)

						if !ok {
							log.Debug("not commit message, skip further processing,tags:",tags,",",ctx.Response.String())
							releaseCtx(ctx)
							return nil
						}
					}

					releaseCtx(ctx)

					offset = pop.NextOffset
				}

				if time.Since(lastCommitTime) > commitIdle {
					//commit on idle timeout
					if !offset.Equals(initOfffset) {
						ok, err := queue.CommitOffset(qConfig, consumer, offset)
						lastCommitTime = time.Now()
						log.Debugf("%v,%v commit offset:%v", qConfig.Name, consumer.Name, offset)
						if !ok || err != nil {
							return err
						}
						initOfffset = offset
					}
				}

				log.Infof("success replay %v messages from queue:[%v,%v], elapsed:%v",len(messages), qConfig.ID,qConfig.Name, time.Since(start1))
			}


			if timeout || len(messages) == 0 {
				log.Debugf("exit flow_runner, [%v][%v] %v messages, timeout:%v", qConfig.Name, consumer.Name, len(messages), timeout)
				return nil
			}

		}
	}

	return nil
}
