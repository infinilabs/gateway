package filters

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestUrlPathFilter struct {
	RequestFilterBase
}

func (filter RequestUrlPathFilter) Name() string {
	return "request_path_filter"
}

func (filter RequestUrlPathFilter) Process(ctx *fasthttp.RequestCtx) {

	path := string(ctx.Path())

	//TODO check cache first

	if global.Env().IsDebug {
		log.Debug("path:", path)
	}

	var hasOtherRules = false
	var hasRules = false
	var valid = false
	valid, hasRules = filter.CheckMustNotRules(path, ctx)
	if !valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	valid, hasRules = filter.CheckMustRules(path, ctx)

	if !valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must rules not matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	var hasShouldRules = false
	valid, hasShouldRules = filter.CheckShouldRules(path, ctx)
	if !valid {
		if !hasOtherRules && hasShouldRules {
			ctx.Filtered()
			if global.Env().IsDebug {
				log.Debugf("only should rules, but none of them are matched, this request has been filtered: %v", ctx.Request.URI().String())
			}
		}
	}

}

