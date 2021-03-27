package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type RequestMethodFilter struct {
	RequestFilterBase
}

func (filter RequestMethodFilter) Name() string {
	return "request_method_filter"
}

func (filter RequestMethodFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	method := string(ctx.Method())

	if global.Env().IsDebug {
		log.Debug("method:", method)
	}

	exclude, ok := filter.GetStringArray("exclude")
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
	if ok {
		for _, x := range exclude {
			if global.Env().IsDebug {
				log.Debugf("exclude method: %v vs %v, match: %v", x, method, util.ToString(x) == method)
			}
			if util.ToString(x) == method {
				filter.Filter(ctx)
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	include, ok := filter.GetStringArray("include")
	if ok {
		for _, x := range include {
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
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}
	if global.Env().IsDebug {
		log.Debug("include:", exclude)
	}

}

