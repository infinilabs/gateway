package throttle

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type DropFilter struct {
	param.Parameters
}

func (filter *DropFilter) Name() string {
	return "drop"
}

func (filter *DropFilter) Filter(ctx *fasthttp.RequestCtx) {
	ctx.Finished()
}


func NewDropFilter(c *config.Config) (pipeline.Filter, error) {
	runner := DropFilter{}
	return &runner, nil
}