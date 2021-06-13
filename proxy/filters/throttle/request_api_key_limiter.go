package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestAPIKeyLimitFilter struct {
	RequestLimiterBase
}

func (filter RequestAPIKeyLimitFilter) Name() string {
	return "request_api_key_limiter"
}

func (filter RequestAPIKeyLimitFilter) Process(ctx *fasthttp.RequestCtx) {

	exists,apiID,_:=ctx.ParseAPIKey()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("api not exist")
		}
		return
	}

	ips, ok := filter.GetStringArray("id")

	apiIDStr :=string(apiID)
	if global.Env().IsDebug {
		log.Trace("api rules: ", len(ips), ", api: ", apiIDStr)
	}

	if ok&&len(ips) > 0 {
		for _, v := range ips {
			if v == apiIDStr {
				if global.Env().IsDebug {
					log.Debug(apiIDStr, "met check rules")
				}
				filter.internalProcess("api", apiIDStr,ctx)
				return
			}
		}
		return
	}

	filter.internalProcess("api", apiIDStr,ctx)
}
