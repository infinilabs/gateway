/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"math/rand"
)

type RatioRoutingFlowFilter struct {
	Ratio              float32 `config:"ratio"`
	Flow               string  `config:"flow"`
	ContinueAfterMatch bool    `config:"continue"`
	flow               common.FilterFlow
}

func (filter *RatioRoutingFlowFilter) Name() string {
	return "ratio"
}

func (filter *RatioRoutingFlowFilter) Filter(ctx *fasthttp.RequestCtx) {

	v := int(filter.Ratio * 100)
	r :=rand.Intn(100)

	if global.Env().IsDebug {
		log.Debugf("split traffic, check [%v] of [%v], hit: %v", r, v, r <= v)
	}

	if r <= v {
		ctx.Request.Header.Set("X-Ratio-Hit","true")
		ctx.Resume()
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s]", ctx.URI().String(), filter.Flow)
		}
		filter.flow.Process(ctx)
		if !filter.ContinueAfterMatch {
			ctx.Finished()
		}
	}else{
		ctx.Request.Header.Set("X-Ratio-Hit","false")
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("ratio",NewRatioRoutingFlowFilter,&RatioRoutingFlowFilter{})
}

func NewRatioRoutingFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RatioRoutingFlowFilter{
		Ratio: 0.1,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.flow = common.MustGetFlow(runner.Flow)

	return &runner, nil
}
