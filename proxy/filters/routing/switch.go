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
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type SwitchFlowFilter struct {
	PathRules          []SwitchRule `config:"path_rules"`
	RemovePrefix       bool         `config:"remove_prefix"`
	ContinueAfterMatch bool         `config:"continue"`
	Unescape           bool         `config:"unescape"`
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
	var err error
	path := string(ctx.RequestURI())
	paths := strings.Split(path, "/")
	indexPart := paths[1]

	if util.ContainStr(indexPart, "%") && filter.Unescape {
		indexPart, err = url.PathUnescape(indexPart)
		if err != nil {
			panic(err)
		}
	}

	for _, item := range filter.PathRules {

		if strings.HasPrefix(indexPart, item.Prefix) {
			if filter.RemovePrefix {
				nexIndex := strings.TrimPrefix(indexPart, item.Prefix)
				//log.Debugf("index:%v, prefix:%v, new index: %v", indexPart, item.Prefix, nexIndex)
				paths[1] = nexIndex
				ctx.Request.SetRequestURI(strings.Join(paths, "/"))
			}

			flow := common.MustGetFlow(item.Flow)
			if global.Env().IsDebug {
				log.Debugf("request [%v] go on flow: [%s]", ctx.PhantomURI().String(), flow.ToString())
			}
			flow.Process(ctx)
			if !filter.ContinueAfterMatch {
				ctx.Finished()
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("switch", NewSwitchFlowFilter, &SwitchFlowFilter{})
}

func NewSwitchFlowFilter(c *config.Config) (pipeline.Filter, error) {
	runner := SwitchFlowFilter{
		RemovePrefix: true,
		Unescape:     true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
