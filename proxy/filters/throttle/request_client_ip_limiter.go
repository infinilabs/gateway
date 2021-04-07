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
	RequestLimiterBase
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

	if ok&&len(ips) > 0 {
		for _, v := range ips {
			//check if ip pre-defined
			if v == clientIP {
				if global.Env().IsDebug {
					log.Debug(clientIP, "met check rules")
				}
				filter.internalProcess("clientIP",clientIP,filterCfg,ctx)
				return
			}
		}
		//terminate if no ip matches
		return
	}

	filter.internalProcess("clientIP",clientIP,filterCfg,ctx)
}


type RequestLimiterBase struct {
	param.Parameters
}

func (filter RequestLimiterBase) internalProcess(tokenType,token string,filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx){

	maxQps, ok := filter.GetInt64("max_requests", -1)
	burstQps, _ := filter.GetInt64("burst_requests", -1)
	maxBps, ok1 := filter.GetInt64("max_bytes", -1)
	burstBps, _ := filter.GetInt64("burst_bytes", -1)
	interval := filter.GetDurationOrDefault("interval", "1s")

	if global.Env().IsDebug{
		log.Trace(ok,",max_requests:",maxQps,",",ok1,",max_bytes:",maxBps,",burst_requests:",burstQps,",burst_bytes:",burstBps,",interval:",interval)
	}

	if ok  || ok1 {
		retryTimes:=0
		maxRetryTimes:=filter.GetIntOrDefault("max_retry_times",1000)
		retryDeplyInMs:=time.Duration(filter.GetIntOrDefault("retry_delay_in_ms",10))*time.Millisecond

	RetryRateLimit:
		if (ok && !rate.GetRateLimiter(filter.UUID()+"_limit_requests", token, int(maxQps),int(burstQps),interval).Allow()) ||
			(ok1 && !rate.GetRateLimiter(filter.UUID()+"_limit_bytes", token, int(maxBps),int(burstBps),interval).AllowN(time.Now(),ctx.Request.GetRequestLength())) {

			if global.Env().IsDebug {
				log.Warn(tokenType," ",token, " reached limit")
			}

			if filter.GetStringOrDefault("action","retry") == "drop"{
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
