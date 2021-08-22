package offline_processing

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"runtime"
	"sync"
	"time"
)

type RunnerConfig struct {
	FlowName   string `config:"flow"`
	InputQueue string `config:"input_queue"`
}

var ctxPool = &sync.Pool{
	New: func() interface{} {
		c := fasthttp.RequestCtx{
			SequenceID: util.GetIncrementID("ctx"),
		}
		return &c
	},
}

func acquireCtx() (ctx *fasthttp.RequestCtx) {
	x1 := ctxPool.Get().(*fasthttp.RequestCtx)
	x1.SequenceID = util.GetIncrementID("ctx")
	x1.Reset()
	return x1
}

func releaseCtx(ctx *fasthttp.RequestCtx) {
	ctx.Reset()
	ctxPool.Put(ctx)
}

type FlowRunner struct {
	param.Parameters
}

var signalChannel chan bool = make(chan bool, 1)

func (this FlowRunner) Stop() error {
	signalChannel <- true
	return nil
}

func (this FlowRunner) Setup(cfg *config.Config) {
}

func (this FlowRunner) Name() string {
	return "flow_runner"
}


func (this FlowRunner) Process(c *pipeline.Context) error {
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
				log.Error("error in FlowRunner,", v)
			}
		}
	}()

	if !this.GetBool("enabled",true){
		return nil
	}

	runnerConfig := RunnerConfig{}
	runnerConfig.InputQueue=this.MustGetString("input_queue")
	runnerConfig.FlowName=this.MustGetString("flow")

	timeOut := 5
	idleDuration := time.Duration(timeOut) * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()
	idleTimeout1 := time.NewTimer(idleDuration)
	defer idleTimeout1.Stop()

	processor := common.GetFlowProcess(runnerConfig.FlowName)

	READ_DOCS:
		stop := false
		for {
			select {
			case <-signalChannel:
				stop = true
				return nil
			default:
				idleTimeout1.Reset(idleDuration)
				if !stop {
					pop,timeout,err := queue.PopTimeout(runnerConfig.InputQueue,idleDuration)
					if err!=nil{
						log.Error(err)
						panic(err)
					}
					if timeout{
						if global.Env().IsDebug {
							log.Tracef("%v no message input", idleDuration)
						}
						goto READ_DOCS
					}
						ctx := acquireCtx()
						err = ctx.Request.Decode(pop)
						if err != nil {
							log.Error(err)
							panic(err)
						}

						processor(ctx)

						releaseCtx(ctx)
				}
			}

		}

	return nil
}
