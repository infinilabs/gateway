package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type RequestMethodFilter struct {
	genericFilter *RequestFilter
	Include       []string `config:"include"`
	Exclude       []string `config:"exclude"`
}

func (filter *RequestMethodFilter) Name() string {
	return "request_method_filter"
}

func init() {
	pipeline.RegisterFilterPlugin("request_method_filter",NewRequestMethodFilter)
}

func NewRequestMethodFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestMethodFilter{}
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

func (filter *RequestMethodFilter) Filter(ctx *fasthttp.RequestCtx) {

	method := string(ctx.Method())

	if global.Env().IsDebug {
		log.Debug("method:", method, ",exclude:", filter.Exclude)
	}

	if len(filter.Exclude) > 0 {
		for _, x := range filter.Exclude {
			if global.Env().IsDebug {
				log.Debugf("exclude method: %v vs %v, match: %v", x, method, util.ToString(x) == method)
			}
			if util.ToString(x) == method {
				filter.genericFilter.Filter(ctx)
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	if len(filter.Include) > 0 {
		for _, x := range filter.Include {
			if global.Env().IsDebug {
				log.Debugf("include method: %v vs %v, match: %v", x, method, util.ToString(x) == string(method))
			}
			if util.ToString(x) == method {
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
