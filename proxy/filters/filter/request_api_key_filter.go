package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestAPIKeyFilter struct {
	genericFilter *RequestFilter
}

func (filter *RequestAPIKeyFilter) Name() string {
	return "request_api_key_filter"
}

func NewRequestAPIKeyFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestAPIKeyFilter {
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.genericFilter= &RequestFilter {
		Action: "deny",
		Status:403,
	}

	if err := c.Unpack(runner.genericFilter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *RequestAPIKeyFilter) Filter(ctx *fasthttp.RequestCtx) {
	exists,apiID,_:=ctx.ParseAPIKey()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("API not exist")
		}
		return
	}

	apiIDStr :=string(apiID)
	valid, hasRule:= filter.genericFilter.CheckExcludeStringRules(apiIDStr, ctx)
	if hasRule&&!valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.genericFilter.CheckIncludeStringRules(apiIDStr, ctx)
	if hasRule&&!valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}

