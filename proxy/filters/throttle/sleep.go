package throttle

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type SleepFilter struct {
	SleepInMs int `config:"sleep_in_million_seconds"`
}

func (filter *SleepFilter) Name() string {
	return "sleep"
}

func (filter *SleepFilter) Filter(ctx *fasthttp.RequestCtx) {
	if filter.SleepInMs <= 0 {
		return
	}
	time.Sleep(time.Duration(filter.SleepInMs) * time.Millisecond)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("sleep",NewSleepFilter,&SleepFilter{})
}

func NewSleepFilter(c *config.Config) (pipeline.Filter, error) {

	runner := SleepFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
