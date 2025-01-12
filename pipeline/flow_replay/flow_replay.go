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

package flow_replay

import (
	"errors"
	"fmt"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/bytebufferpool"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type Config struct {
	QueueField   param.ParaKey `config:"queue_name_field"`
	MessageField param.ParaKey `config:"message_field"`

	MessageIncludeResponse         bool   `config:"message_include_response"`
	KeepTags                       bool   `config:"keep_tags"`
	FlowName                       string `config:"flow"`
	FlowMaxRunningTimeoutInSeconds int    `config:"flow_max_running_timeout_in_second"`
	CommitOnTag                    string `config:"commit_on_tag"`
	IdleWaitTimeoutInSeconds       int    `config:"idle_wait_timeout_in_second"`
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
	pipeline.RegisterProcessorPlugin("flow_replay", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		QueueField:                     "queue_name",
		MessageField:                   "messages",
		CommitOnTag:                    "",
		IdleWaitTimeoutInSeconds:       1,
		FlowMaxRunningTimeoutInSeconds: 60,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	runner := FlowRunnerProcessor{config: &cfg}
	return &runner, nil
}

func (processor FlowRunnerProcessor) Stop() error {
	signalChannel <- true
	return nil
}

func (processor *FlowRunnerProcessor) Name() string {
	return "flow_replay"
}

func (processor *FlowRunnerProcessor) Process(ctx *pipeline.Context) error {

	if global.Env().IsDebug {
		log.Debugf("start flow_replay [%v]", processor.config.FlowName)
		defer log.Debugf("exit flow_replay [%v]", processor.config.FlowName)
	}

	//get message from queue
	queueName := ctx.Get(processor.config.QueueField)
	obj := ctx.Get(processor.config.MessageField)
	if obj != nil {
		flowProcessor := common.GetFlowProcess(processor.config.FlowName)
		start1 := time.Now()

		messages := obj.([]queue.Message)
		if global.Env().IsDebug {
			log.Tracef("get %v messages from context", len(messages))
		}

		if len(messages) == 0 {
			return nil
		}
		//parse template
		mainBuf := bytebufferpool.Get("flow_replay")
		defer bytebufferpool.Put("flow_replay", mainBuf)
		var err error
		for _, pop := range messages {

			stats.Increment("flow_replay", "message_received")

			if global.ShuttingDown() {
				return errors.New("shutting down")
			}

			filterCtx := acquireCtx()

			if processor.config.MessageIncludeResponse {
				err = filterCtx.Decode(pop.Data)
			} else {
				err = filterCtx.Request.Decode(pop.Data)
			}

			if err != nil {
				log.Error(err)
				panic(err)
			}

			if global.Env().IsDebug {
				log.Tracef("start forward request to flow:%v", processor.config.FlowName)
			}

			filterCtx.SetFlowID(processor.config.FlowName)

			filterCtx.Set("MESSAGE_QUEUE_NAME", queueName)
			filterCtx.Set("MESSAGE_OFFSET", pop.Offset)
			filterCtx.Set("NEXT_MESSAGE_OFFSET", pop.NextOffset)

			flowProcessor(filterCtx)

			if processor.config.KeepTags {
				tags, ok := filterCtx.GetTags()
				if ok {
					ts := []string{}
					for _, v := range tags {
						ts = append(ts, v)
					}
					ctx.AddTags(ts)
				}
			}

			if global.Env().IsDebug {
				log.Tracef("end forward request to flow:%v", processor.config.FlowName)
			}

			if processor.config.CommitOnTag != "" {
				tags, ok := filterCtx.GetTags()
				if ok {
					_, ok = tags[processor.config.CommitOnTag]
				}

				if !ok {
					return errors.New("commit tag was not found")
				}

				if !ok {
					releaseCtx(filterCtx)
					return nil
				}
				stats.Increment("flow_replay", "message_succeed")
			}
			releaseCtx(filterCtx)
		}

		log.Debugf("replay %v messages flow:[%v], elapsed:%v", len(messages), processor.config.FlowName, time.Since(start1))

	}

	return nil
}
