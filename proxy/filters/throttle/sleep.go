package throttle

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type SleepFilter struct {
	param.Parameters
}

func (filter SleepFilter) Name() string {
	return "sleep"
}

func (filter SleepFilter) Process(ctx *fasthttp.RequestCtx) {
	sleepInMs,ok:=filter.GetInt64("sleep_in_million_seconds",1000)
	if !ok{
		return
	}
	time.Sleep(time.Duration(sleepInMs)*time.Millisecond)
}


