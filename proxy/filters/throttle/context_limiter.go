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

package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ContextLimitFilter struct {
	limiter *GenericLimiter
	Context []string `config:"context"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_limiter", NewContextLimitFilter, &ContextLimitFilter{})
}

func NewContextLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ContextLimitFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	limiter := genericLimiter
	runner.limiter = &limiter

	if err := c.Unpack(runner.limiter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.limiter.init()

	return &runner, nil
}

func (filter *ContextLimitFilter) Name() string {
	return "context_limiter"
}

func (filter *ContextLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("context rules: ", len(filter.Context))
	}

	if len(filter.Context) > 0 {
		data := []string{}
		for _, v := range filter.Context {
			x, err := ctx.GetValue(v)
			if err != nil {
				log.Debugf("context:%v,%v,%v", v, x, err)
			} else {
				data = append(data, util.ToString(x))
			}
		}
		if len(data) > 0 {
			filter.limiter.internalProcess("context", util.JoinArray(data, ","), ctx)
		}
	}
}
