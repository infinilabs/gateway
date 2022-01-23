package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ContextLimitFilter struct {
	limiter *GenericLimiter
	Context []string `config:"context"`
}

func NewContextLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ContextLimitFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	limiter := genericLimiter
	runner.limiter = &limiter

	if err := c.Unpack(runner.limiter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.limiter.init()

	return &runner, nil
}

func (filter *ContextLimitFilter) Name() string {
	return "context_limiter"
}

func (filter *ContextLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("context rules: ", len(filter.Context))
	}

	if len(filter.Context) > 0 {
		data := []string{}
		for _, v := range filter.Context {
			x, err := ctx.GetValue(v)
			if err != nil {
				log.Debugf("context:%v,%v,%v", v, x, err)
			} else {
				data = append(data, util.ToString(x))
			}
		}
		if len(data) > 0 {
			filter.limiter.internalProcess("context", util.JoinArray(data, ","), ctx)
		}
	}
}
