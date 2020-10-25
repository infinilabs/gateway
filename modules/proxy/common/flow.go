package common

import (
	"infini.sh/framework/core/errors"
	"infini.sh/framework/lib/fasthttp"
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

type RoutingRule struct {
	Method      string
	PathPattern string
	FlowID      string
}

type FilterFlow struct {
	FilterName string
	Filters []func(ctx *fasthttp.RequestCtx)
}

func NewFilterFlow(name string,filters ...func(ctx *fasthttp.RequestCtx)) FilterFlow {
	flow:=FilterFlow{FilterName:name,Filters: filters}
	return flow
}

func (flow FilterFlow)JoinFlows(flows ...FilterFlow) FilterFlow {
	for _,v:=range flows{
		for _,x:=range v.Filters{
			flow.Filters=append(flow.Filters,x)
		}
	}
	return flow
}

func (flow FilterFlow) Name() string{
	return flow.FilterName
}

func (flow FilterFlow)Process(ctx *fasthttp.RequestCtx){
	for _,v:=range flow.Filters{
		v(ctx)
	}
}

func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flow:=GetFlow(flowID)
	return flow.Process
}

func GetFlow(flowID string) FilterFlow {
	 v,ok:=flows[flowID]
	 if ok{
	 	return v
	 }
	 panic(errors.New("flow was not found"))
}

func JoinFlows(flowID ...string) FilterFlow {
	flow:=FilterFlow{}
	for _,v:=range flowID{
		temp:=GetFlow(v)
		flow.JoinFlows(flow,temp)
	}
	return flow
}

func GetFilter(filterID string) func(ctx *fasthttp.RequestCtx) {
	return filters[filterID]
}

var filters map[string]func(ctx *fasthttp.RequestCtx) = make(map[string]func(ctx *fasthttp.RequestCtx))
var flows map[string]FilterFlow = make(map[string]FilterFlow)

func RegisterFilter(filter Filter) {
	filters[filter.Name()] = filter.Process
}

func RegisterFlow(flow FilterFlow) {
	flows[flow.Name()] = flow
}
