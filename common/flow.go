package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"reflect"
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
				log.Debugf("filter [%v] not continued", v.Name())
			}
			ctx.AddFlowProcess("skipped")
			break
		}
		if global.Env().IsDebug {
			log.Debugf("processing filter [%v] [%v]", v.Name(), v)
		}
		ctx.AddFlowProcess(v.Name())
		v.Filter(ctx)
	}
}

func MustGetFlow(flowID string) FilterFlow {

	if flowID==""{
		panic("flow id can't be nil")
	}

	v, ok := flows[flowID]
	if ok {
		return v
	}

	v = FilterFlow{}
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

	flows[flowID] = v
	return v
}

var flowLock sync.RWMutex

func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flowLock.Lock()
	defer flowLock.Unlock()
	flow := MustGetFlow(flowID)
	return flow.Process
}

//get filter plugins
func GetFilter(name string) pipeline.Filter {
	v, ok := filterPluginTypes[name]
	if !ok {
		panic(errors.Errorf("filter [%s] not found", name))
	}
	return v
}

func GetFilterInstanceWithConfig(cfg *FilterConfig) pipeline.Filter {
	if global.Env().IsDebug {
		log.Tracef("get filter instance [%v] [%v]", cfg.Name, cfg.ID)
	}

	if cfg.ID == "" {
		panic(errors.Errorf("invalid filter config [%v] [%v] is not set", cfg.Name, cfg.ID))
	}

	v1, ok := filterInstances[cfg.ID]
	if ok {
		if global.Env().IsDebug {
			log.Debugf("hit filter instance [%v] [%v], return", cfg.Name, cfg.ID)
		}
		return v1
	}

	if cfg.Name == "" {
		panic(errors.Errorf("the type of filter [%v] [%v] is not set", cfg.Name, cfg.ID))
	}

	filter := GetFilter(cfg.Name)
	t := reflect.ValueOf(filter).Type()
	v := reflect.New(t).Elem()

	f := v.FieldByName("Data")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Map {
		f.Set(reflect.ValueOf(cfg.Parameters))
	}
	x := v.Interface().(pipeline.Filter)
	filterInstances[cfg.ID] = x
	return x
}

func GetFilterInstanceWithConfigV2(filterName string, cfg *config.Config) pipeline.Filter {
	if global.Env().IsDebug {
		log.Debugf("get filter [%v]", filterName)
	}

	if !cfg.HasField("_meta:config_id") {
		panic(errors.Errorf("invalid filter config [%v] [%v] is not set", filterName, cfg))
	}
	id, err := cfg.String("_meta:config_id", -1)
	if err != nil {
		panic(err)
	}

	v1, ok := filterInstances[id]
	if ok {
		if global.Env().IsDebug {
			log.Debugf("hit filter instance [%v] [%v], return", filterName, id)
		}
		return v1
	}

	////check contional
	//if cfg.HasField("when"){
	//
	//}

	parameters := map[string]interface{}{}
	cfg.Unpack(&parameters)

	filter := GetFilter(filterName)
	t := reflect.ValueOf(filter).Type()
	v := reflect.New(t).Elem()

	f := v.FieldByName("Data")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Map {
		f.Set(reflect.ValueOf(parameters))
	}
	x := v.Interface().(pipeline.Filter)

	filterInstances[id] = x
	return x
}

var filterPluginTypes map[string]pipeline.Filter = make(map[string]pipeline.Filter)

var filterInstances map[string]pipeline.Filter = make(map[string]pipeline.Filter)
var flows map[string]FilterFlow = make(map[string]FilterFlow)

var routingRules map[string]RuleConfig = make(map[string]RuleConfig)
var flowConfigs map[string]FlowConfig = make(map[string]FlowConfig)
var routerConfigs map[string]RouterConfig = make(map[string]RouterConfig)

func RegisterFilterPlugin(filter pipeline.Filter) {
	log.Debug("register filter: ", filter.Name())
	filterPluginTypes[filter.Name()] = filter
}

func RegisterFlowConfig(flow FlowConfig) {
	flowConfigs[flow.Name] = flow
}

func RegisterRouterConfig(config RouterConfig) {
	routerConfigs[config.Name] = config
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
