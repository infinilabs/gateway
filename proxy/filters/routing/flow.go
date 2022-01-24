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

type FlowFilter struct {
	Flows []string `config:"flows"`
}

func (filter *FlowFilter) Name() string {
	return "flow"
}

func (filter *FlowFilter) Filter(ctx *fasthttp.RequestCtx) {
	for _, v := range filter.Flows {
		flow := common.MustGetFlow(v)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), v, flow.ToString())
		}
		ctx.AddFlowProcess("flow:" + flow.ID)
		flow.Process(ctx)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("flow",NewFlowFilter,&FlowFilter{})
}

func NewFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := FlowFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
