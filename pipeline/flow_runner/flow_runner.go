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
	FlowName   string `config:"flow"`
	InputQueue string `config:"input_queue"`
	FlowMaxRunningTimeoutInSeconds int `config:"flow_max_running_timeout_in_second"`

	CommitOnTag   	 string `config:"commit_on_tag"`
	IdleWaitTimeoutInSeconds   	 int `config:"idle_wait_timeout_in_second"`

	Consumer   queue.ConsumerConfig `config:"consumer"`

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
	return x1
}

func releaseCtx(ctx *fasthttp.RequestCtx) {
	ctx.Reset()
	ctxPool.Put(ctx)
}

type FlowRunnerProcessor struct {
	config *Config
}

var signalChannel = make(chan bool, 1)


func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		Consumer: queue.ConsumerConfig{
			Group: "group-001",
			Name: "consumer-001",
			FetchMinBytes:   	1,
			FetchMaxMessages:   100,
			FetchMaxWaitMs:   10000,
		},
		CommitOnTag:"",
		IdleWaitTimeoutInSeconds:   	1,
		FlowMaxRunningTimeoutInSeconds:   	10,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner:= FlowRunnerProcessor{config: &cfg}
	return &runner,nil
}


func (processor FlowRunnerProcessor) Stop() error {
	signalChannel <- true
	return nil
}

func (processor *FlowRunnerProcessor) Name() string {
	return "flow_runner"
}

func (processor *FlowRunnerProcessor) Process(ctx *pipeline.Context) error {
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
				log.Errorf("error in flow_runner [%v], [%v]",processor.config.FlowName, v)
				ctx.Failed()
			}
		}
	}()

	//timeOut := 5
	//idleDuration := time.Duration(timeOut) * time.Second
	flowProcessor := common.GetFlowProcess(processor.config.FlowName)
	qConfig :=queue.GetOrInitConfig(processor.config.InputQueue)

	var consumer=queue.GetOrInitConsumerConfig(qConfig.Id,processor.config.Consumer.Group,processor.config.Consumer.Name)
	var initOfffset string
	var offset string
	for {
			t1:=util.AcquireTimer(time.Duration(processor.config.FlowMaxRunningTimeoutInSeconds)*time.Second)
			defer util.ReleaseTimer(t1)

			if ctx.IsCanceled(){
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
				if global.Env().IsDebug{
					log.Debug(qConfig.Name,",",consumer.Group,",",consumer.Name,",init offset:",offset)
				}
				wg:=sync.WaitGroup{}
				wg.Add(1)
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
								log.Error("error on run tasks,", v)
							}
						}
						wg.Done()
					}()

					//READ_DOCS:
					initOfffset,_=queue.GetOffset(qConfig,consumer)
					offset=initOfffset

					_,messages,timeout,err:=queue.Consume(qConfig,consumer.Name,offset,processor.config.Consumer.FetchMaxMessages,time.Millisecond*time.Duration(processor.config.Consumer.FetchMaxWaitMs))

					if len(messages) > 0 {

						for _,pop:=range messages {
							if err!=nil{
								log.Error(err)
								panic(err)
							}

							ctx := acquireCtx()
							err = ctx.Request.Decode(pop.Data)
							if err != nil {
								log.Error(err)
								panic(err)
							}

							flowProcessor(ctx)

							if processor.config.CommitOnTag!=""{
								tags,ok:=ctx.GetTags()
								if ok{
									_,ok= tags[processor.config.CommitOnTag]
								}
								if !ok{
									releaseCtx(ctx)
									return
								}
							}

							releaseCtx(ctx)

							offset=pop.NextOffset
							if offset!=""&&offset!=initOfffset{
								ok,err:=queue.CommitOffset(qConfig,consumer,offset)
								log.Tracef("%v,%v commit offset:%v",qConfig.Name,consumer.Name,offset)
								if !ok||err!=nil{
									panic(err)
								}
							}
						}
					}

					if timeout||len(messages)==0{
						time.Sleep(1*time.Second)
						//goto READ_DOCS
					}

					}()
				wg.Wait()
				return nil
			}
		}

	return nil
}
