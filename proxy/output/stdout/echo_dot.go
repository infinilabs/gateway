/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package stdout

import (
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type EchoDot struct {
	pipeline.Parameters
}

func (filter EchoDot) Name() string {
	return "echo_dot"
}

func (joint EchoDot) Process(ctx *fasthttp.RequestCtx) {
	size:=joint.GetIntOrDefault("repeat",1)
	for i:=0;i<size;i++{
		ctx.WriteString(".")
	}
}
