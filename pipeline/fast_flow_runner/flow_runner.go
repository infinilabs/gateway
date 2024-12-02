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

package fast_flow_runner

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
	FlowName                       string               `config:"flow"`
	InputQueue                     string               `config:"input_queue"`
	NumOfWorkers                   int                  `config:"worker_size"`
	FlowMaxRunningTimeoutInSeconds int                  `config:"flow_max_running_timeout_in_second"`
	Consumer                       queue.ConsumerConfig `config:"consumer"`
}

var ctxPool = &sync.Pool{
	New: func() interface{} {
		c := fasthttp.RequestCtx{}
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
	wg     sync.WaitGroup
}

var signalChannel = make(chan bool, 1)

func init() {
	pipeline.RegisterProcessorPlugin("fast_flow_runner", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		NumOfWorkers: 1,
		Consumer: queue.ConsumerConfig{
			Group:             "group-001",
			Name:              "consumer-001",
			FetchMinBytes:     1,
			FetchMaxBytes:     20 * 1024 * 1024,
			FetchMaxMessages:  500,
			EOFRetryDelayInMs: 500,
			FetchMaxWaitMs:    10000,
			ConsumeTimeoutInSeconds:         60,
			EOFMaxRetryTimes:         10,
			ClientExpiredInSeconds: 60,
		},
		FlowMaxRunningTimeoutInSeconds: 60,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := FlowRunnerProcessor{config: &cfg}
	runner.wg = sync.WaitGroup{}
	return &runner, nil
}

func (processor FlowRunnerProcessor) Stop() error {
	signalChannel <- true
	return nil
}

func (processor *FlowRunnerProcessor) Name() string {
	return "fast_flow_runner"
}

func (processor *FlowRunnerProcessor) Process(ctx *pipeline.Context) error {

	if processor.config.NumOfWorkers <= 0 {
		processor.config.NumOfWorkers = 1
	}
	for i := 0; i < processor.config.NumOfWorkers; i++ {
		processor.wg.Add(1)
		go processor.HandleQueueConfig(ctx)
	}

	processor.wg.Wait()

	return nil
}

func (processor *FlowRunnerProcessor) HandleQueueConfig(ctx *pipeline.Context) error {

	defer func() {

		processor.wg.Done()

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
				log.Errorf("error in disorder_flow_runner [%v], [%v]", processor.config.FlowName, v)
				ctx.Failed(fmt.Errorf("fast flow runner panic: %v", r))
			}
		}

	}()

	qConfig := queue.GetOrInitConfig(processor.config.InputQueue)
	flowProcessor := common.GetFlowProcess(processor.config.FlowName)

	t1 := util.AcquireTimer(time.Duration(processor.config.FlowMaxRunningTimeoutInSeconds) * time.Second)
	defer util.ReleaseTimer(t1)

	for {
		if ctx.IsCanceled() {
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
			messages, timeout, err := queue.PopTimeout(qConfig, time.Duration(processor.config.Consumer.FetchMaxWaitMs)*time.Millisecond)
			if err != nil {
				log.Tracef("error on queue:[%v]", qConfig.Name)
				panic(err)
			}
			if err != nil && err.Error() != "EOF" {
				log.Error(err)
				panic(err)
			}

			if len(messages) > 0 {
				ctx := acquireCtx()
				err = ctx.Request.Decode(messages)
				if err != nil {
					log.Error(err)
					panic(err)
				}
				ctx.SetFlowID(processor.config.FlowName)
				flowProcessor(ctx)
				releaseCtx(ctx)
			}

			if timeout || len(messages) == 0 {
				return nil
			}

		}
	}
}
