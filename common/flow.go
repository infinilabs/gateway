package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/lib/fasthttp"
	"reflect"
	"strings"
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

//func NewFilterFlow(name string, filters ...func(ctx *fasthttp.RequestCtx)) FilterFlow {
//	flow := FilterFlow{FlowName: name, Filters: filters}
//	return flow
//}

func (flow *FilterFlow) JoinFilter(filter ...RequestFilter) *FilterFlow {
	for _,v:=range filter{
		flow.Filters=append(flow.Filters,v)
	}
	return flow
}

func (flow *FilterFlow) JoinFlows(flows ...FilterFlow) *FilterFlow {
	for _, v := range flows {
		for _, x := range v.Filters {
			flow.Filters = append(flow.Filters, x)
		}
	}
	return flow
}

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
		v.Process(ctx)
	}
}

func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flow := GetFlow(flowID)
	return flow.Process
}

func GetFlow(flowID string) FilterFlow {
	v, ok := flows[flowID]
	if ok {
		return v
	}
	panic(errors.New("flow was not found"))
}

func JoinFlows(flowID ...string) FilterFlow {
	flow := FilterFlow{}
	for _, v := range flowID {
		temp := GetFlow(v)
		flow.JoinFlows(flow, temp)
	}
	return flow
}

func GetFilter(name string) RequestFilter {
	return filters[name]
}



func GetFilterWithConfig(cfg *FilterConfig) RequestFilter {
	log.Tracef("get filter instances, %v", cfg.Name)
	if filters[cfg.Name] != nil {
		t := reflect.ValueOf(filters[cfg.Name]).Type()
		v := reflect.New(t).Elem()

		f := v.FieldByName("Data")
		if f.IsValid() && f.CanSet() && f.Kind() == reflect.Map {
			f.Set(reflect.ValueOf(cfg.Parameters))
		}
		return v.Interface().(RequestFilter)
	}
	panic(errors.New(cfg.Name + " not found"))
}

var filters map[string]RequestFilter = make(map[string]RequestFilter)
var flows map[string]FilterFlow = make(map[string]FilterFlow)

var filterConfigs map[string]FilterConfig = make(map[string]FilterConfig)
var routingRules map[string]RoutingRule = make(map[string]RoutingRule)
var flowConfigs map[string]FlowConfig = make(map[string]FlowConfig)
var routerConfigs map[string]RouterConfig = make(map[string]RouterConfig)

func RegisterFilter(filter RequestFilter) {
	filters[filter.Name()] = filter
}

func RegisterFlow(flow FilterFlow) {
	flows[flow.ID] = flow
}

func RegisterFlowConfig(flow FlowConfig) {
	flowConfigs[flow.Name] = flow
}

func RegisterRoutingRule(rule RoutingRule) {
	routingRules[rule.ID] = rule
}
func RegisterRouterConfig(config RouterConfig) {
	routerConfigs[config.Name] = config
}

func GetRouter(name string) RouterConfig {
	return routerConfigs[name]
}

func GetRule(name string) RoutingRule {
	return routingRules[name]
}

func GetFlowConfig(name string) FlowConfig {
	return flowConfigs[name]
}
