/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package echo

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type EchoDot struct {
	param.Parameters
}

func (filter EchoDot) Name() string {
	return "echo"
}

func (filter EchoDot) Process(ctx *fasthttp.RequestCtx) {
	str:=filter.GetStringOrDefault("str",".")
	size:=filter.GetIntOrDefault("repeat",1)
	for i:=0;i<size;i++{
		ctx.WriteString(str)
	}
	ctx.Finished()
}
