package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"strings"
	"sync"
)

type FilterFlow struct {
	orm.ORMObjectBase
	Filters []pipeline.Filter
}

func (flow *FilterFlow) JoinFilter(filter pipeline.Filter) *FilterFlow {
	flow.Filters = append(flow.Filters, filter)
	return flow
}

func (flow *FilterFlow) ToString() string {
	str := strings.Builder{}
	has := false
	for _, v := range flow.Filters {
		if has {
			str.WriteString(" > ")
		}
		str.WriteString(v.Name())
		has = true
	}
	return str.String()
}

func (flow *FilterFlow) Process(ctx *fasthttp.RequestCtx) {
	for _, v := range flow.Filters {
		if !ctx.ShouldContinue() {
			if global.Env().IsDebug {
				log.Tracef("filter [%v] not continued", v.Name())
			}
			ctx.AddFlowProcess("skipped")
			break
		}
		if global.Env().IsDebug {
			log.Tracef("processing filter [%v] [%v]", v.Name(), v)
		}
		ctx.AddFlowProcess(v.Name())
		v.Filter(ctx)
	}
}

func MustGetFlow(flowID string) FilterFlow {

	if flowID==""{
		panic("flow id can't be nil")
	}

	v1,ok:=flows.Load(flowID)
	if ok{
		return v1.(FilterFlow)
	}

	v := FilterFlow{}
	cfg := GetFlowConfig(flowID)

	if global.Env().IsDebug {
		log.Tracef("flow [%v] [%v]", flowID, cfg)
	}

	if len(cfg.Filters) > 0 {
		flow1, err := pipeline.NewFilter(cfg.GetConfig())
		if err != nil {
			panic(err)
		}
		v.JoinFilter(flow1)
	}

	flows.Store(flowID, v)
	return v
}

func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flow := MustGetFlow(flowID)
	return flow.Process
}

var flows = &sync.Map{}

var routingRules map[string]RuleConfig = make(map[string]RuleConfig)
var flowConfigs map[string]FlowConfig = make(map[string]FlowConfig)
var routerConfigs map[string]RouterConfig = make(map[string]RouterConfig)

func init() {
	//api.HandleAPIMethod("GET","entry", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//
	//})

}

func ClearFlowCache(flow string) {
	flows.Delete(flow)
}

func RegisterFlowConfig(flow FlowConfig) {

	if flow.ID == "" && flow.Name != "" {
		flow.ID = flow.Name
	}

	flowConfigs[flow.ID] = flow
	flowConfigs[flow.Name] = flow
	ClearFlowCache(flow.ID)
	ClearFlowCache(flow.Name)
}

func RegisterRouterConfig(config RouterConfig) {
	if config.ID == "" && config.Name != "" {
		config.ID = config.Name
	}
	routerConfigs[config.ID] = config
}

func GetRouter(name string) RouterConfig {
	v, ok := routerConfigs[name]
	if !ok {
		panic(errors.Errorf("router [%s] not found", name))
	}
	return v
}

func GetFlowConfig(id string) FlowConfig {
	v, ok := flowConfigs[id]
	if !ok {
		panic(errors.Errorf("flow [%s] not found", id))
	}
	return v
}
