package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"time"
)

type RequestClientIPLimitFilter struct {
	param.Parameters
}

func (filter RequestClientIPLimitFilter) Name() string {
	return "request_client_ip_limiter"
}

func (filter RequestClientIPLimitFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	ips, ok := filter.GetStringArray("ip")

	clientIP := ctx.RemoteIP().String()

	if global.Env().IsDebug {
		log.Trace("ips rules: ", len(ips), ", client_ip: ", clientIP)
	}

	if len(ips) > 0 {
		for _, v := range ips {
			if v == clientIP {
				if global.Env().IsDebug {
					log.Debug(clientIP, "met check rules")
				}
				goto CHECK
			}
		}
		//terminate if no ip matches
		return
	}

CHECK:

	maxQps, ok := filter.GetInt64("max_qps", -1)
	maxBps, ok1 := filter.GetInt64("max_bps", -1)

	if ok  || ok1 {
		retryTimes:=0
		maxRetryTimes:=filter.GetIntOrDefault("max_retry",1000)
		retryDeplyInMs:=time.Duration(filter.GetIntOrDefault("retry_delay_in_ms",10))*time.Millisecond

		RetryRateLimit:
		if (ok && !rate.GetRaterWithDefine(filter.UUID()+"_max_qps", clientIP, int(maxQps)).Allow()) || (ok1 && !rate.GetRaterWithDefine(filter.UUID()+"_max_bps", clientIP, int(maxBps)).AllowN(time.Now(),ctx.Request.GetRequestLength())) {

			if global.Env().IsDebug {
				log.Warn("client_ip ",clientIP, " reached limit")
			}

			if filter.GetStringOrDefault("action","retry") == "deny"{
				ctx.SetStatusCode(429)
				ctx.WriteString(filter.GetStringOrDefault("message", "Reach request limit!"))
				ctx.Finished()
				return
			}else{
				if retryTimes>maxRetryTimes{
					ctx.SetStatusCode(429)
					ctx.WriteString(filter.GetStringOrDefault("message", "Retried but still beyond request limit!"))
					ctx.Finished()
					return
				}
				time.Sleep(retryDeplyInMs)
				retryTimes++
				goto RetryRateLimit
			}
		}
	}

}
