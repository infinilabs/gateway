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
	uuid           string
	MaxRequests    int    `config:"max_requests"`
	BurstRequests  int    `config:"burst_requests"`
	MaxBytes       int    `config:"max_bytes"`
	BurstBytes     int    `config:"burst_bytes"`
	Interval       string `config:"interval"`
	Action         string `config:"action"`
	MaxRetryTimes  int    `config:"max_retry_times"`
	RetryDelayInMs int    `config:"retry_delay_in_ms"`
	Status         int    `config:"status"`
	Message        string `config:"message"`
	WarnMessage    bool `config:"log_warn_message"`
	RetriedMessage string `config:"failed_retry_message"`

	interval       time.Duration
	retryDeplyInMs time.Duration
}

var genericLimiter = GenericLimiter{
	MaxRequests:    -1,
	BurstRequests:  -1,
	MaxBytes:       -1,
	BurstBytes:     -1,
	MaxRetryTimes:  1000,
	RetryDelayInMs: 10,
	Interval:       "1s",
	Action:         "retry",
	Status:         429,
	Message:        "Reach request limit!",
	RetriedMessage: "Retried but still beyond request limit!",
}

func (filter *GenericLimiter) init() {
	filter.uuid = util.GetUUID()
	filter.retryDeplyInMs = time.Duration(filter.RetryDelayInMs) * time.Millisecond
	filter.interval = util.GetDurationOrDefault(filter.Interval, 1*time.Second)
}

func (filter *GenericLimiter) internalProcess(tokenType, token string, ctx *fasthttp.RequestCtx) {
	filter.internalProcessWithValues(tokenType,token,ctx,1,ctx.Request.GetRequestLength())
}

func (filter *GenericLimiter) internalProcessWithValues(tokenType, token string, ctx *fasthttp.RequestCtx,hits, bytes int) {

	if global.Env().IsDebug {
		log.Tracef("limit config: %v, type:%v, token:%v", filter,tokenType,token)
	}

	if filter.MaxRequests > 0 || filter.MaxBytes > 0 {
		retryTimes := 0
	RetryRateLimit:
		if (filter.MaxRequests > 0 && !rate.GetRateLimiter(filter.uuid+"_limit_requests", token, int(filter.MaxRequests), int(filter.BurstRequests), filter.interval).AllowN(time.Now(),hits)) ||
			(filter.MaxBytes > 0 && !rate.GetRateLimiter(filter.uuid+"_limit_bytes", token, int(filter.MaxBytes), int(filter.BurstBytes), filter.interval).AllowN(time.Now(), bytes)) {

			if global.Env().IsDebug {
				log.Warn(tokenType, " ", token, " reached limit")
			}

			if filter.MaxRequests > 0 &&filter.MaxRequests<hits{
				log.Warn(tokenType, " ", token, " reached limit: ",filter.MaxRequests," by:", hits,", seems the limit is too small")
			}

			if filter.MaxBytes > 0 &&filter.MaxBytes<bytes{
				log.Warn(tokenType, " ", token, " reached limit: ",filter.MaxBytes," by:", bytes,", seems the limit is too small")
			}

			if filter.Action == "drop" {
				ctx.SetStatusCode(filter.Status)
				ctx.WriteString(filter.Message)

				if filter.WarnMessage{
					log.Warnf("request throttled: %v, %v %v",tokenType,token,string(ctx.Path()))
				}

				ctx.Finished()
				return
			} else {
				if retryTimes > filter.MaxRetryTimes {
					ctx.SetStatusCode(filter.Status)
					ctx.WriteString(filter.RetriedMessage)

					if filter.WarnMessage{
						log.Warnf("request throttled: %v %v %v",tokenType,token,string(ctx.Path()))
					}

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
