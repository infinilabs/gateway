package filter

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	log "src/github.com/cihub/seelog"
)

type DeadlockCheckFilter struct {
	param.Parameters
}

func (filter DeadlockCheckFilter) Name() string {
	return "deadlock_checker"
}

const RetryKey = "RETRIED_TIMES"

func (filter DeadlockCheckFilter) Process(ctx *fasthttp.RequestCtx) {

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
		queue.Push(filter.MustGetString("deadlock_queue"),ctx.Request.Encode())
		return
	}

	times++
	ctx.Request.Header.Set(RetryKey,util.IntToString(times))
}

