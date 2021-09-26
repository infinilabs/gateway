package throttle

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type DropFilter struct {
	param.Parameters
}

func (filter DropFilter) Name() string {
	return "drop"
}

func (filter DropFilter) Process(ctx *fasthttp.RequestCtx) {
	ctx.Finished()
}