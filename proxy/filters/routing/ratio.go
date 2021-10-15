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
	"sync"
)

type RatioRoutingFlowFilter struct {
	randPool           *sync.Pool
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

	seeds := filter.randPool.Get().(*rand.Rand)
	defer filter.randPool.Put(seeds)

	r := seeds.Intn(100)

	if global.Env().IsDebug {
		log.Debugf("split traffic, check [%v] of [%v]", r, v)
	}

	if r <= v {
		ctx.Resume()
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s]", ctx.URI().String(), filter.Flow)
		}
		filter.flow.Process(ctx)
		if !filter.ContinueAfterMatch {
			ctx.Finished()
		}
	}

}

func NewRatioRoutingFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RatioRoutingFlowFilter{
		Ratio: 0.1,
		randPool: &sync.Pool{
			New: func() interface{} {
				return rand.New(rand.NewSource(100))
			},
		},
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.flow = common.MustGetFlow(runner.Flow)

	return &runner, nil
}
