package throttle

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"time"
)

type SleepFilter struct {
	param.Parameters
}

func (filter SleepFilter) Name() string {
	return "sleep"
}

func (filter SleepFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
	sleepInMs,ok:=filter.GetInt64("sleep_in_million_seconds",-1)
	if !ok{
		return
	}
	time.Sleep(time.Duration(sleepInMs)*time.Millisecond)
}
