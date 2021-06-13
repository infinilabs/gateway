package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestAPIKeyFilter struct {
	RequestFilterBase
}

func (filter RequestAPIKeyFilter) Name() string {
	return "request_api_key_filter"
}

func (filter RequestAPIKeyFilter) Process(ctx *fasthttp.RequestCtx) {
	exists,apiID,_:=ctx.ParseAPIKey()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("API not exist")
		}
		return
	}

	apiIDStr :=string(apiID)
	valid, hasRule:= filter.CheckExcludeStringRules(apiIDStr, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(apiIDStr, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}

