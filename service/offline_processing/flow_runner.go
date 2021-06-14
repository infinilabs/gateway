package offline_processing

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"runtime"
	"sync"
	"time"
)

type RunnerConfig struct {
	Enabled    bool   `config:"enabled"`
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
	Config *RunnerConfig
}

var signalChannel chan bool

func (this FlowRunner) Start() error {

	signalChannel = make(chan bool, 1)

	runnerConfig := RunnerConfig{}
	ok, err := env.ParseConfig("flow_runner", &runnerConfig)
	if ok && err != nil {
		panic(err)
	}

	if runnerConfig.Enabled == false {
		return nil
	}

	timeOut := 5
	idleDuration := time.Duration(timeOut) * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()
	idleTimeout1 := time.NewTimer(idleDuration)
	defer idleTimeout1.Stop()

	processor := common.GetFlowProcess(runnerConfig.FlowName)

	go func() {
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

	READ_DOCS:
		stop := false
		for {
			select {
			case <-signalChannel:
				stop = true
				return
			default:
				idleTimeout1.Reset(idleDuration)
				if !stop {
					select {

					case pop := <-queue.ReadChan(runnerConfig.InputQueue):
						ctx := acquireCtx()
						err := ctx.Request.Decode(pop)
						if err != nil {
							log.Error(err)
							panic(err)
						}

						processor(ctx)

						releaseCtx(ctx)

					case <-idleTimeout1.C:
						if global.Env().IsDebug {
							log.Tracef("%v no message input", idleDuration)
						}
						goto READ_DOCS
					}
				}
			}

		}
	}()

	return nil
}

func (this FlowRunner) Stop() error {
	signalChannel <- true
	return nil
}

func (this FlowRunner) Setup(cfg *config.Config) {
}

func (this FlowRunner) Name() string {
	return "flow_runner"
}
