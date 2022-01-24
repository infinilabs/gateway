package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestHostLimitFilter struct {
	limiter *GenericLimiter
	Host    []string `config:"host"`
}

func init() {
	pipeline.RegisterFilterPlugin("request_host_limiter",NewRequestHostLimitFilter)
}

func NewRequestHostLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestHostLimitFilter{}

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

func (filter *RequestHostLimitFilter) Name() string {
	return "request_host_limiter"
}

func (filter *RequestHostLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	hostStr := string(ctx.Host())

	if global.Env().IsDebug {
		log.Trace("host rules: ", len(filter.Host), ", host: ", hostStr)
	}

	if len(filter.Host) > 0 {
		for _, v := range filter.Host {
			if v == hostStr {
				if global.Env().IsDebug {
					log.Debug(hostStr, "met check rules")
				}
				filter.limiter.internalProcess("host", hostStr, ctx)
				return
			}
		}
		return
	}

	filter.limiter.internalProcess("host", hostStr, ctx)
}
