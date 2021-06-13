package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
)

type RequestClientIPLimitFilter struct {
	RequestLimiterBase
}

func (filter RequestClientIPLimitFilter) Name() string {
	return "request_client_ip_limiter"
}

func (filter RequestClientIPLimitFilter) Process(ctx *fasthttp.RequestCtx) {

	ips, ok := filter.GetStringArray("ip")

	clientIP := ctx.RemoteIP().String()

	if global.Env().IsDebug {
		log.Trace("ips rules: ", len(ips), ", client_ip: ", clientIP)
	}

	if ok&&len(ips) > 0 {
		for _, v := range ips {
			//check if ip pre-defined
			if v == clientIP {
				if global.Env().IsDebug {
					log.Debug(clientIP, "met check rules")
				}
				filter.internalProcess("clientIP",clientIP,ctx)
				return
			}
		}
		//terminate if no ip matches
		return
	}

	filter.internalProcess("clientIP",clientIP,ctx)
}
