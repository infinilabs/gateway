package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestHostLimitFilter struct {
	RequestLimiterBase
}

func (filter RequestHostLimitFilter) Name() string {
	return "request_host_limiter"
}

func (filter RequestHostLimitFilter) Process(ctx *fasthttp.RequestCtx) {

	hostStr :=string(ctx.Host())

	hosts, ok := filter.GetStringArray("host")

	if global.Env().IsDebug {
		log.Trace("host rules: ", len(hosts), ", host: ", hostStr)
	}

	if ok&&len(hosts) > 0 {
		for _, v := range hosts {
			if v == hostStr {
				if global.Env().IsDebug {
					log.Debug(hostStr, "met check rules")
				}
				filter.internalProcess("host", hostStr,ctx)
				return
			}
		}
		return
	}

	filter.internalProcess("host", hostStr,ctx)
}
