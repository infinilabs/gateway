package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestAPIKeyLimitFilter struct {
	limiter *GenericLimiter
	APIKeys []string `config:"id"`
}

func NewRequestAPIKeyLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestAPIKeyLimitFilter{}
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

func (filter *RequestAPIKeyLimitFilter) Name() string {
	return "request_api_key_limiter"
}

func (filter *RequestAPIKeyLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	exists, apiID, _ := ctx.ParseAPIKey()
	if !exists {
		if global.Env().IsDebug {
			log.Tracef("api not exist")
		}
		return
	}

	apiIDStr := string(apiID)
	if global.Env().IsDebug {
		log.Trace("api rules: ", len(filter.APIKeys), ", api: ", apiIDStr)
	}

	if len(filter.APIKeys) > 0 {
		for _, v := range filter.APIKeys {
			if v == apiIDStr {
				if global.Env().IsDebug {
					log.Debug(apiIDStr, "met check rules")
				}
				filter.limiter.internalProcess("api", apiIDStr, ctx)
				return
			}
		}
		return
	}

	filter.limiter.internalProcess("api", apiIDStr, ctx)
}
