package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type ResponseHeaderFilter struct {
	RequestFilterBase
}

func (filter ResponseHeaderFilter) Name() string {
	return "response_header_filter"
}

func (filter ResponseHeaderFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Debug("headers:", string(util.EscapeNewLine(ctx.Response.Header.Header())))
	}

	exclude, ok := filter.GetMapArray("exclude")
	if ok {
		for _, x := range exclude {
			for k, v := range x {
				v1 := ctx.Response.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("exclude header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					filter.Filter(ctx)
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
	}

	include, ok := filter.GetMapArray("include")
	if ok {
		for _, x := range include {
			for k, v := range x {
				v1 := ctx.Response.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("include header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
		filter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}

}

