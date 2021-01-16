package filters

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseStatusCodeFilter struct {
	RequestFilterBase
}

func (filter ResponseStatusCodeFilter) Name() string {
	return "response_status_filter"
}

func (filter ResponseStatusCodeFilter) Process(ctx *fasthttp.RequestCtx) {

	code := ctx.Response.StatusCode()

	if global.Env().IsDebug {
		log.Debug("code:", code)
	}

	exclude, ok := filter.GetInt64Array("exclude")
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
	if ok {
		for _, x := range exclude {
			y:=int(x)
			if global.Env().IsDebug {
				log.Debugf("exclude code: %v vs %v, match: %v", x, code,  y== code)
			}
			if y == code {
				ctx.Filtered()
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	include, ok := filter.GetInt64Array("include")
	if global.Env().IsDebug {
		log.Debug("include:", exclude)
	}
	if ok {
		for _, x := range include {
			y:=int(x)
			if global.Env().IsDebug {
				log.Debugf("include code: %v vs %v, match: %v", x, code, y == code)
			}
			if y == code {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
				}
				return
			}
		}
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}


}

