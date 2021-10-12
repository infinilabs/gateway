package queue

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
)

type StatsFilter struct {
	Category string `config:"category"`
}

func (filter StatsFilter) Name() string {
	return "stats"
}

func (filter StatsFilter) Filter(ctx *fasthttp.RequestCtx) {

	stats.Timing(filter.Category,"response.elapsed_ms",ctx.GetElapsedTime().Milliseconds())
	stats.IncrementBy(filter.Category,"response.bytes", int64(ctx.Response.GetResponseLength()))
	stats.Increment(filter.Category,fmt.Sprintf("response.status.%v",ctx.Response.StatusCode()))

	stats.IncrementBy(filter.Category,"request.bytes", int64(ctx.Request.GetRequestLength()))
	stats.Increment(filter.Category,fmt.Sprintf("request.method.%v",string(ctx.Request.Header.Method())))

}

func NewStatsFilter(c *config.Config) (pipeline.Filter, error) {

	runner := StatsFilter{
		Category: global.Env().GetAppLowercaseName(),
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}