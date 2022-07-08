package flow_runner

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"runtime"
	"sync"
	"time"
)

type Config struct {
	FlowName                       string `config:"flow"`
	InputQueue                     string `config:"input_queue"`
	FlowMaxRunningTimeoutInSeconds int    `config:"flow_max_running_timeout_in_second"`
	CommitTimeoutInSeconds         int    `config:"commit_timeout_in_second"`

	CommitOnTag              string `config:"commit_on_tag"`
	IdleWaitTimeoutInSeconds int    `config:"idle_wait_timeout_in_second"`

	Consumer queue.ConsumerConfig `config:"consumer"`
}

var ctxPool = &sync.Pool{
	New: func() interface{} {
		c := fasthttp.RequestCtx{
			//SequenceID: util.GetIncrementID("ctx"),
		}
		return &c
	},
}

func acquireCtx() (ctx *fasthttp.RequestCtx) {
	x1 := ctxPool.Get().(*fasthttp.RequestCtx)
	//x1.SequenceID = util.GetIncrementID("ctx")
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
			Group:            "group-001",
			Name:             "consumer-001",
			FetchMinBytes:    1,
			FetchMaxBytes:    10 * 1024 * 1024,
			FetchMaxMessages: 500,
			FetchMaxWaitMs:   1000,
		},
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
	var initOfffset string
	var offset string
	qConfig := queue.GetOrInitConfig(processor.config.InputQueue)
	flowProcessor := common.GetFlowProcess(processor.config.FlowName)

	if !queue.HasLag(qConfig) {
		return nil
	}

	var consumer = queue.GetOrInitConsumerConfig(qConfig.Id, processor.config.Consumer.Group, processor.config.Consumer.Name)
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
				log.Errorf("error in flow_runner [%v], [%v]", processor.config.FlowName, v)
				ctx.Failed()
				skipFinalDocsProcess = true
			}
		}

		if skipFinalDocsProcess {
			return
		}

		if offset != "" && offset != initOfffset {
			ok, err := queue.CommitOffset(qConfig, consumer, offset)
			log.Tracef("%v,%v commit offset:%v", qConfig.Name, consumer.Name, offset)
			if !ok || err != nil {
				ctx.Failed()
			}
		}
	}()

	t1 := util.AcquireTimer(time.Duration(processor.config.FlowMaxRunningTimeoutInSeconds) * time.Second)
	defer util.ReleaseTimer(t1)

	initOfffset, _ = queue.GetOffset(qConfig, consumer)
	offset = initOfffset
	lastCommitTime := time.Now()
	var commitIdle = time.Duration(processor.config.CommitTimeoutInSeconds) * time.Second
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
			if global.Env().IsDebug {
				log.Debug(qConfig.Name, ",", consumer.Group, ",", consumer.Name, ",init offset:", offset)
			}

			log.Tracef("star to consume queue:%v", qConfig.Name)
			ctx1, messages, timeout, err := queue.Consume(qConfig, consumer, offset)
			log.Tracef("get %v messages from queue:%v", len(messages), qConfig.Name)

			if err != nil && err.Error() != "EOF" {
				log.Error(err)
				panic(err)
			}

			if len(messages) > 0 {
				for _, pop := range messages {
					ctx := acquireCtx()
					err = ctx.Request.Decode(pop.Data)
					if err != nil {
						log.Error(err)
						panic(err)
					}

					ctx.SetFlowID(processor.config.FlowName)

					flowProcessor(ctx)

					if processor.config.CommitOnTag != "" {
						tags, ok := ctx.GetTags()
						if ok {
							_, ok = tags[processor.config.CommitOnTag]
						}
						if !ok {
							releaseCtx(ctx)
							return nil
						}
					}

					releaseCtx(ctx)

					offset = pop.NextOffset

				}

				if time.Since(lastCommitTime) > commitIdle {
					//commit on idle timeout
					if offset != "" && offset != initOfffset {
						ok, err := queue.CommitOffset(qConfig, consumer, offset)
						lastCommitTime = time.Now()
						log.Tracef("%v,%v commit offset:%v", qConfig.Name, consumer.Name, offset)
						if !ok || err != nil {
							return err
						}
					}
				}

			}
			offset = ctx1.NextOffset

			if timeout || len(messages) == 0 {
				log.Debugf("[%v][%v] %v messages, timeout:%v, sleep 1s", qConfig.Name, consumer.Name, len(messages), timeout)
				return nil
			}

		}
	}

	return nil
}
