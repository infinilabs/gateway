/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type GenericLimiter struct {
	uuid string
	MaxRequests int    `config:"max_requests"`
	BurstRequests int    `config:"burst_requests"`
	MaxBytes int    `config:"max_bytes"`
	BurstBytes int    `config:"burst_bytes"`
	Interval string    `config:"interval"`
	Action string    `config:"action"`
	MaxRetryTimes int    `config:"max_retry_times"`
	RetryDelayInMs int    `config:"retry_delay_in_ms"`
	Message string    `config:"message"`
	RetriedMessage string    `config:"failed_retry_message"`

	interval time.Duration
	retryDeplyInMs time.Duration
}

var genericLimiter = GenericLimiter{
	MaxRequests:-1,
	BurstRequests:-1,
	MaxBytes:-1,
	BurstBytes:-1,
	MaxRetryTimes:1000,
	RetryDelayInMs:10,
	Interval:"1s",
	Action:"retry",
	Message:"Reach request limit!",
	RetriedMessage:"Retried but still beyond request limit!",
}

func (filter *GenericLimiter) init(){
	filter.uuid=util.GetUUID()
	filter.retryDeplyInMs=time.Duration(filter.RetryDelayInMs)*time.Millisecond
	filter.interval=util.GetDurationOrDefault(filter.Interval,1*time.Second)
}

func (filter *GenericLimiter) internalProcess(tokenType,token string,ctx *fasthttp.RequestCtx){

	if global.Env().IsDebug{
		log.Tracef("limit config: %v",filter)
	}

	if filter.MaxRequests>0||filter.MaxBytes>0 {
		retryTimes:=0
	RetryRateLimit:
		if (filter.MaxRequests>0 && !rate.GetRateLimiter(filter.uuid+"_limit_requests", token, int(filter.MaxRequests),int(filter.BurstRequests),filter.interval).Allow()) ||
			(filter.MaxBytes>0 && !rate.GetRateLimiter(filter.uuid+"_limit_bytes", token, int(filter.MaxBytes),int(filter.BurstBytes),filter.interval).AllowN(time.Now(),ctx.Request.GetRequestLength())) {

			if global.Env().IsDebug {
				log.Warn(tokenType," ",token, " reached limit")
			}

			if filter.Action== "drop"{
				ctx.SetStatusCode(429)
				ctx.WriteString(filter.Message)
				ctx.Finished()
				return
			}else{
				if retryTimes>filter.MaxRetryTimes{
					ctx.SetStatusCode(429)
					ctx.WriteString(filter.RetriedMessage)
					ctx.Finished()
					return
				}
				time.Sleep(filter.retryDeplyInMs)
				retryTimes++
				goto RetryRateLimit
			}
		}
	}
}

