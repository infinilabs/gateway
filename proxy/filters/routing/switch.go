/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	log "src/github.com/cihub/seelog"
	"strings"
)

type SwitchFlowFilter struct {
	param.Parameters
}

func (filter SwitchFlowFilter) Name() string {
	return "switch"
}

type SwitchRule struct {

}

func (filter SwitchFlowFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
	v,ok:=filter.GetMapArray("path_rules")
	if !ok{
		return
	}

	path:=string(ctx.RequestURI())
	paths:=strings.Split(path,"/")
	indexPart:= paths[1]
	continueAfterMatch := filter.GetBool("continue", false)

	for _,item:=range v{
		prefix:=item["prefix"].(string)
		if strings.HasPrefix(indexPart,prefix){
			//removePrefix:=item["remove_prefix"].(bool)
			//if removePrefix{
				nexIndex:=strings.TrimLeft(indexPart,prefix)
				paths[1]=nexIndex
				ctx.Request.SetRequestURI(strings.Join(paths,"/"))
				flowName:=item["flow"].(string)
				flow:=common.MustGetFlow(flowName)
				if global.Env().IsDebug{
					log.Debugf("request [%v] go on flow: [%s]",ctx.URI().String(),flow.ToString())
				}
				flow.Process(ctx)
				if !continueAfterMatch {
					ctx.Finished()
				}
			//}
		}
	}
}
