package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type RetryLimiter struct {
	param.Parameters
}

func (filter RetryLimiter) Name() string {
	return "retry_limiter"
}

const RetryKey = "Retried_times"

func (filter RetryLimiter) Process(ctx *fasthttp.RequestCtx) {

	timeBytes:=ctx.Request.Header.Peek(RetryKey)
	times:=0
	if timeBytes!=nil{
		t,err:=util.ToInt(string(timeBytes))
		if err==nil{
			times=t
		}
	}

	if times>filter.GetIntOrDefault("max_retry_times",3){
		log.Debugf("hit max retry times")
		ctx.Finished()
		ctx.Request.Header.Del(RetryKey)
		queue.Push(filter.MustGetString("queue_name"),ctx.Request.Encode())
		return
	}

	times++
	ctx.Request.Header.Set(RetryKey,util.IntToString(times))
}

