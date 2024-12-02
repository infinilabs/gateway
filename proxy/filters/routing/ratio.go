// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"fmt"
	"math/rand"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type RatioRoutingFlowFilter struct {
	Ratio              float32 `config:"ratio"`
	Flow               string  `config:"flow"`
	Action             string  `config:"action"` //redirect_flow or drop
	ContinueAfterMatch bool    `config:"continue"`
	flow               common.FilterFlow
}

func (filter *RatioRoutingFlowFilter) Name() string {
	return "ratio"
}

func (filter *RatioRoutingFlowFilter) Filter(ctx *fasthttp.RequestCtx) {

	v := int(filter.Ratio * 100)
	r := rand.Intn(100)

	if global.Env().IsDebug {
		log.Tracef("split traffic, check [%v] of [%v], hit: %v", r, v, r <= v)
	}

	if r < v {
		ctx.Request.Header.Set("X-Ratio-Hit", "true")
		if filter.Action == redirectAction {
			ctx.Resume()
			if global.Env().IsDebug {
				log.Tracef("request [%v] go on flow: [%s]", ctx.PhantomURI().String(), filter.Flow)
			}
			filter.flow.Process(ctx)
			if !filter.ContinueAfterMatch {
				ctx.Finished()
			}
		} else {
			ctx.Finished()
		}
	} else {
		ctx.Request.Header.Set("X-Ratio-Hit", "false")
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("ratio", NewRatioRoutingFlowFilter, &RatioRoutingFlowFilter{})
}

const redirectAction = "redirect_flow"
const dropAction = "drop"

func NewRatioRoutingFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RatioRoutingFlowFilter{
		Action: redirectAction,
		Ratio:  0.1,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.flow = common.MustGetFlow(runner.Flow)

	return &runner, nil
}
