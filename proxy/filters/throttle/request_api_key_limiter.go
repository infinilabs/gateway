package throttle


import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type RequestAPIKeyLimitFilter struct {
	RequestLimiterBase
}

func (filter RequestAPIKeyLimitFilter) Name() string {
	return "request_api_key_limiter"
}

func (filter RequestAPIKeyLimitFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

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
				filter.internalProcess("api", apiIDStr,filterCfg,ctx)
				return
			}
		}
		return
	}

	filter.internalProcess("api", apiIDStr,filterCfg,ctx)
}
