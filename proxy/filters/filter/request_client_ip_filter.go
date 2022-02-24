package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestClientIPFilter struct {
	genericFilter *RequestFilter
	Include       []string `config:"include"`
	Exclude       []string `config:"exclude"`
}

func (filter *RequestClientIPFilter) Name() string {
	return "request_client_ip_filter"
}

func (filter *RequestClientIPFilter) Filter(ctx *fasthttp.RequestCtx) {

	clientIP := ctx.RemoteIP().String()

	if global.Env().IsDebug{
		log.Trace("client_ip:",clientIP)
	}

	valid, hasRule := CheckExcludeStringRules(clientIP, filter.Exclude, ctx)
	if hasRule && !valid {
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		filter.genericFilter.Filter(ctx)
		return
	}

	valid, hasRule = CheckIncludeStringRules(clientIP, filter.Include, ctx)
	if hasRule && !valid {
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		filter.genericFilter.Filter(ctx)
		return
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_client_ip_filter",NewRequestClientIPFilter,&RequestAPIKeyFilter{})
}

func NewRequestClientIPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestClientIPFilter{}
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
