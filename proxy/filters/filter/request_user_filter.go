package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestUserFilter struct {
	RequestFilterBase
}

func (filter RequestUserFilter) Name() string {
	return "request_user_filter"
}

func (filter RequestUserFilter) Process(ctx *fasthttp.RequestCtx) {
	exists,user,_:=ctx.ParseBasicAuth()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("user not exist")
		}
		return
	}

	userStr:=string(user)
	valid, hasRule:= filter.CheckExcludeStringRules(userStr, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(userStr, ctx)
	if hasRule&&!valid {
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}

