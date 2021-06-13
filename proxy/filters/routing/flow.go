package routing

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	log "github.com/cihub/seelog"
)

type FlowFilter struct {
	param.Parameters
}

func (filter FlowFilter) Name() string {
	return "flow"
}

func (filter FlowFilter) Process(ctx *fasthttp.RequestCtx) {
	flows := filter.MustGetStringArray("flows")
	for _, v := range flows {
		flow := common.MustGetFlow(v)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), v, flow.ToString())
		}
		ctx.AddFlowProcess("flow:"+flow.ID)
		flow.Process(ctx)
	}
}

//type IfElseThenFilter struct {
//	param.Parameters
//}
//
//func (filter IfElseThenFilter) Name() string {
//	return "if"
//}
//
//func (filter IfElseThenFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
//
//	flows := filter.MustGetStringArray("flows")
//	for _, v := range flows {
//		flow := common.MustGetFlow(v)
//		if global.Env().IsDebug {
//			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), v, flow.ToString())
//		}
//		flow.Process(ctx)
//	}
//}
