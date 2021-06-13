package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestClientIPFilter struct {
	RequestFilterBase
}

func (filter RequestClientIPFilter) Name() string {
	return "request_client_ip_filter"
}

func (filter RequestClientIPFilter) Process(ctx *fasthttp.RequestCtx) {

	clientIP:=ctx.RemoteIP().String()

	valid, hasRule:= filter.CheckExcludeStringRules(clientIP, ctx)
	if hasRule&&!valid {
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		filter.Filter(ctx)
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(clientIP, ctx)
	if hasRule&&!valid {
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		filter.Filter(ctx)
		return
	}

}

