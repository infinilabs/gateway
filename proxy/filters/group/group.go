/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package group

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type GroupFilter struct {
	param.Parameters
}

func (filter GroupFilter) Name() string {
	return "group"
}

func (filter GroupFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

}
