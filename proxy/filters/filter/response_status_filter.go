package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseStatusCodeFilter struct {
	genericFilter *RequestFilter
	Include       []int `config:"include"`
	Exclude       []int `config:"exclude"`
}

func (filter ResponseStatusCodeFilter) Name() string {
	return "response_status_filter"
}

func NewResponseStatusCodeFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ResponseStatusCodeFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.genericFilter = &RequestFilter{
		Action: "deny",
		Status: 403,
	}

	if err := c.Unpack(runner.genericFilter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *ResponseStatusCodeFilter) Filter(ctx *fasthttp.RequestCtx) {

	code := ctx.Response.StatusCode()

	if global.Env().IsDebug {
		log.Debug("code:", code, ",exclude:", filter.Exclude)
	}
	if len(filter.Exclude) > 0 {
		for _, x := range filter.Exclude {
			y := int(x)
			if global.Env().IsDebug {
				log.Debugf("exclude code: %v vs %v, match: %v", x, code, y == code)
			}
			if y == code {
				filter.genericFilter.Filter(ctx)
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	if global.Env().IsDebug {
		log.Debug("include:", filter.Include)
	}
	if len(filter.Include) > 0 {
		for _, x := range filter.Include {
			y := int(x)
			if global.Env().IsDebug {
				log.Debugf("include code: %v vs %v, match: %v", x, code, y == code)
			}
			if y == code {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
				}
				return
			}
		}
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}
}
