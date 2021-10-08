package queue

import (
	"fmt"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
)

type StatsFilter struct {
	param.Parameters
}

func (filter StatsFilter) Name() string {
	return "stats"
}

func (filter StatsFilter) Process(ctx *fasthttp.RequestCtx) {

	stats.Timing(filter.GetStringOrDefault("category","gateway"),"response.elapsed_ms",ctx.GetElapsedTime().Milliseconds())
	stats.IncrementBy(filter.GetStringOrDefault("category","gateway"),"request.bytes", int64(ctx.Request.GetRequestLength()))
	stats.IncrementBy(filter.GetStringOrDefault("category","gateway"),"response.bytes", int64(ctx.Response.GetResponseLength()))
	stats.Increment(filter.GetStringOrDefault("category","gateway"),fmt.Sprintf("response.%v",ctx.Response.StatusCode()))

}

