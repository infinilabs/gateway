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
)

type CloneFlowFilter struct {
	Flows    []string `config:"flows"`
	Continue bool     `config:"continue"`
}

func (filter *CloneFlowFilter) Name() string {
	return "clone"
}

func (filter *CloneFlowFilter) Filter(ctx *fasthttp.RequestCtx) {

	for _, v := range filter.Flows {
		ctx.Resume()
		flow := common.MustGetFlow(v)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.PhantomURI().String(), v, flow.ToString())
		}

		//ctx.UpdateCurrentFlow(flow) //TODO, tracking flow

		flow.Process(ctx)
	}

	if len(filter.Flows) > 0 && !filter.Continue {
		ctx.Finished()
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("clone", NewCloneFlowFilter, &CloneFlowFilter{})
}

func NewCloneFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := CloneFlowFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
