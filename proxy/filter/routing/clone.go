/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	log "github.com/cihub/seelog"
)

type CloneFlowFilter struct {
	param.Parameters
}

func (filter CloneFlowFilter) Name() string {
	return "clone"
}

func (filter CloneFlowFilter) Process(ctx *fasthttp.RequestCtx) {
		flows:=filter.MustGetStringArray("flows")
		for _,v:=range flows{
			ctx.Resume()
			flow:=common.MustGetFlow(v)
			if global.Env().IsDebug{
				log.Debugf("request [%v] go on flow: [%s] [%s]",ctx.URI().String(),v,flow.ToString())
			}
			flow.Process(ctx)
			ctx.Finished()
		}

}
