package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)


type RequestServerHostFilter struct {
	RequestFilterBase
}

func (filter RequestServerHostFilter) Name() string {
	return "request_host_filter"
}

func (filter RequestServerHostFilter) Process(ctx *fasthttp.RequestCtx) {
	host:=string(ctx.Request.Host())
	valid, hasRule:= filter.CheckExcludeStringRules(host, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(host, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}
