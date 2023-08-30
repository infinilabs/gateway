package flow_replay

import (
	"errors"
	"fmt"
	"infini.sh/framework/core/param"
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
	MessageField  param.ParaKey `config:"message_field"`
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

			if global.ShuttingDown(){
				return errors.New("shutting down")
			}

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

				if !ok{
					log.Error("commit tag was not found")
					return errors.New("commit tag was not found")
				}

				if !ok {
					releaseCtx(ctx)
					return nil
				}
			}
			releaseCtx(ctx)
		}

		log.Infof("replay %v messages flow:[%v], elapsed:%v", len(messages), processor.config.FlowName, time.Since(start1))

	}

	return nil
}
