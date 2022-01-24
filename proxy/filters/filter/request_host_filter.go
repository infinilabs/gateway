package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestServerHostFilter struct {
	genericFilter *RequestFilter
	Include       []string `config:"include"`
	Exclude       []string `config:"exclude"`
}

func (filter *RequestServerHostFilter) Name() string {
	return "request_host_filter"
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_host_filter",NewRequestServerHostFilter,&RequestServerHostFilter{})
}

func NewRequestServerHostFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestServerHostFilter{}
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

func (filter *RequestServerHostFilter) Filter(ctx *fasthttp.RequestCtx) {
	host := string(ctx.Request.Host())
	valid, hasRule := CheckExcludeStringRules(host, filter.Exclude, ctx)
	if hasRule && !valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule = CheckIncludeStringRules(host, filter.Include, ctx)
	if hasRule && !valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}
