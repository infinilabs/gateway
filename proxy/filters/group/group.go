/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package group

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type GroupFilter struct {
	param.Parameters
}

func (filter GroupFilter) Name() string {
	return "group"
}

func (filter GroupFilter) Process(ctx *fasthttp.RequestCtx) {

}
