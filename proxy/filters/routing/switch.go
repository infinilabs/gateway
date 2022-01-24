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
	"strings"
)

type SwitchFlowFilter struct {
	PathRules          []SwitchRule `config:"path_rules"`
	RemovePrefix       bool         `config:"remove_prefix"`
	ContinueAfterMatch bool         `config:"continue"`
}

func (filter *SwitchFlowFilter) Name() string {
	return "switch"
}

type SwitchRule struct {
	Prefix string `config:"prefix"`
	Flow   string `config:"flow"`
}

func (filter *SwitchFlowFilter) Filter(ctx *fasthttp.RequestCtx) {
	if len(filter.PathRules) == 0 {
		return
	}

	path := string(ctx.RequestURI())
	paths := strings.Split(path, "/")
	indexPart := paths[1]

	for _, item := range filter.PathRules {
		if strings.HasPrefix(indexPart, item.Prefix) {
			if filter.RemovePrefix {
				nexIndex := strings.TrimLeft(indexPart, item.Prefix)
				paths[1] = nexIndex
				ctx.Request.SetRequestURI(strings.Join(paths, "/"))
				flow := common.MustGetFlow(item.Flow)
				if global.Env().IsDebug {
					log.Debugf("request [%v] go on flow: [%s]", ctx.URI().String(), flow.ToString())
				}
				flow.Process(ctx)
				if !filter.ContinueAfterMatch {
					ctx.Finished()
				}
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPlugin("switch",NewSwitchFlowFilter)
}

func NewSwitchFlowFilter(c *config.Config) (pipeline.Filter, error) {
	runner := SwitchFlowFilter{
		RemovePrefix: true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
