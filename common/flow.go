package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"reflect"
	"strings"
	"sync"
)

/*
# Rules：
METHOD 			PATH 						FLOW
GET 			/							name=cache_first flow =[ get_cache >> forward >> set_cache ]
GET				/_cat/*item					name=forward flow=[forward]
POST || PUT		/:index/_doc/*id			name=forward flow=[forward]
POST || PUT		/:index/_bulk || /_bulk 	name=async_indexing_via_translog flow=[ save_translog ]
POST || GET		/*index/_search				name=cache_first flow=[ get_cache >> forward >> set_cache ]
POST || PUT		/:index/_bulk || /_bulk 	name=async_dual_writes flow=[ save_translog{name=dual_writes_id1, retention=7days, max_size=10gb} ]
POST || PUT		/:index/_bulk || /_bulk 	name=sync_dual_writes flow=[ mirror_forward ]
GET				/audit/*operations			name=secured_audit_access flow=[ basic_auth >> flow{name=cache_first} ]
*/

type FilterFlow struct {
	ID string
	Filters []RequestFilter
}

func (flow *FilterFlow) JoinFilter(filter RequestFilter) *FilterFlow {
	flow.Filters=append(flow.Filters,filter)
	return flow
}

//func (flow *FilterFlow) JoinFlows(flows ...FilterFlow) *FilterFlow {
//	for _, v := range flows {
//		for _, x := range v.Filters {
//			flow.Filters = append(flow.Filters, x)
//		}
//	}
//	return flow
//}

func (flow *FilterFlow) ToString() string {
	str:=strings.Builder{}
	has:=false
	for _,v:=range flow.Filters{
		if has{
			str.WriteString(" > ")
		}
		str.WriteString(v.Name())
		has=true
	}
	return str.String()
}

func (flow *FilterFlow) Process(ctx *fasthttp.RequestCtx) {
	for _, v := range flow.Filters {
		if !ctx.ShouldContinue(){
			if global.Env().IsDebug{
				log.Debugf("filter [%v] not continued",v.Name())
			}
			ctx.AddFlowProcess("skipped")
			break
		}
		if global.Env().IsDebug{
			log.Debugf("processing filter [%v] [%v]",v.Name(),v)
		}
		ctx.AddFlowProcess(v.Name())
		v.Process(ctx)
	}
}

func MustGetFlow(flowID string) FilterFlow {
	v, ok := flows[flowID]
	if ok {
		return v
	}

	v=FilterFlow{}
	cfg:=GetFlowConfig(flowID)

	if global.Env().IsDebug{
		log.Tracef("flow [%v] [%v]",flowID,cfg)
	}

	if len(cfg.FiltersV2)>0{
		flow1, err := New(cfg.FiltersV2)
		if err != nil {
			panic(err)
		}
		v.JoinFilter(flow1)
	}else{
		for _,z:=range cfg.Filters{
			f:= GetFilterInstanceWithConfig(&z)
			v.JoinFilter(f)
		}
	}

	flows[flowID]=v
	return v
}

var flowLock sync.RWMutex
func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flowLock.Lock()
	defer flowLock.Unlock()
	flow := MustGetFlow(flowID)
	return flow.Process
}

//func JoinFlows(flowID ...string) FilterFlow {
//	flow := FilterFlow{}
//	for _, v := range flowID {
//		temp := MustGetFlow(v)
//		flow.JoinFlows(flow, temp)
//	}
//	return flow
//}

//get filter plugins
func GetFilter(name string) RequestFilter {
	v, ok := filterPluginTypes[name]
	if !ok{
		panic(errors.Errorf("filter [%s] not found",name))
	}
	return v
}


func GetFilterInstanceWithConfig(cfg *FilterConfig) RequestFilter {
		if global.Env().IsDebug {
			log.Tracef("get filter instance [%v] [%v]", cfg.Name,cfg.ID)
		}

		if cfg.ID==""{
			panic(errors.Errorf("invalid filter config [%v] [%v] is not set",cfg.Name,cfg.ID))
		}

		v1,ok:=filterInstances[cfg.ID]
		if ok{
			if global.Env().IsDebug{
				log.Debugf("hit filter instance [%v] [%v], return",cfg.Name,cfg.ID)
			}
			return v1
		}

		if cfg.Name==""{
			panic(errors.Errorf("the type of filter [%v] [%v] is not set",cfg.Name,cfg.ID))
		}

		filter:=GetFilter(cfg.Name)
		t := reflect.ValueOf(filter).Type()
		v := reflect.New(t).Elem()

		f := v.FieldByName("Data")
		if f.IsValid() && f.CanSet() && f.Kind() == reflect.Map {
			f.Set(reflect.ValueOf(cfg.Parameters))
		}
		x:= v.Interface().(RequestFilter)
		filterInstances[cfg.ID]=x
		return x
}

func GetFilterInstanceWithConfigV2(filterName string,cfg *config.Config) RequestFilter {
		if global.Env().IsDebug {
			log.Debugf("get filter [%v]", filterName)
		}

		if !cfg.HasField("_meta:config_id"){
			panic(errors.Errorf("invalid filter config [%v] [%v] is not set",filterName,cfg))
		}
		id,err:=cfg.String("_meta:config_id",-1)
		if err!=nil{
			panic(err)
		}

		v1,ok:=filterInstances[id]
		if ok{
			if global.Env().IsDebug{
				log.Debugf("hit filter instance [%v] [%v], return",filterName,id)
			}
			return v1
		}


		////check contional
		//if cfg.HasField("when"){
		//
		//}

		parameters:=map[string]interface{}{}
		cfg.Unpack(&parameters)

		filter:=GetFilter(filterName)
		t := reflect.ValueOf(filter).Type()
		v := reflect.New(t).Elem()

		f := v.FieldByName("Data")
		if f.IsValid() && f.CanSet() && f.Kind() == reflect.Map {
			f.Set(reflect.ValueOf(parameters))
		}
		x:= v.Interface().(RequestFilter)

		filterInstances[id]=x
		return x
}

var filterPluginTypes map[string]RequestFilter = make(map[string]RequestFilter)

var filterInstances map[string]RequestFilter = make(map[string]RequestFilter)
var flows map[string]FilterFlow = make(map[string]FilterFlow)

var routingRules map[string]RuleConfig = make(map[string]RuleConfig)
var flowConfigs map[string]FlowConfig = make(map[string]FlowConfig)
var routerConfigs map[string]RouterConfig = make(map[string]RouterConfig)

func RegisterFilterPlugin(filter RequestFilter) {
	log.Trace("register filter: ",filter.Name())
	filterPluginTypes[filter.Name()] = filter
}

func RegisterFlow(flow FilterFlow) {
	flows[flow.ID] = flow
}

func RegisterFlowConfig(flow FlowConfig) {
	//make sure each filter have dedicated ID
	for i,v:=range flow.Filters{
		if v.ID==""{
			v.ID=util.GetUUID()
			flow.Filters[i]=v
		}
	}

	flowConfigs[flow.Name] = flow
}

func RegisterRoutingRule(rule RuleConfig) {
	routingRules[rule.ID] = rule
}
func RegisterRouterConfig(config RouterConfig) {
	routerConfigs[config.Name] = config
}

func GetRouter(name string) RouterConfig {
	v,ok:=  routerConfigs[name]
	if !ok{
		panic(errors.Errorf("router [%s] not found",name))
	}
	return v
}

func GetRule(id string) RuleConfig {
	v,ok:= routingRules[id]
	if !ok{
		panic(errors.Errorf("rule [%s] not found",id))
	}
	return v
}

func GetFlowConfig(id string) FlowConfig {
	v,ok:= flowConfigs[id]
	if !ok{
		panic(errors.Errorf("flow [%s] not found",id))
	}
	return v
}
