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
	"infini.sh/framework/lib/fasthttp"
)

type RequestHostLimitFilter struct {
	limiter *GenericLimiter
	Host    []string `config:"host"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_host_limiter", NewRequestHostLimitFilter, &RequestHostLimitFilter{})
}

func NewRequestHostLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestHostLimitFilter{}

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

func (filter *RequestHostLimitFilter) Name() string {
	return "request_host_limiter"
}

func (filter *RequestHostLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	hostStr := string(ctx.Host())

	if global.Env().IsDebug {
		log.Trace("host rules: ", len(filter.Host), ", host: ", hostStr)
	}

	if len(filter.Host) > 0 {
		for _, v := range filter.Host {
			if v == hostStr {
				if global.Env().IsDebug {
					log.Debug(hostStr, "met check rules")
				}
				filter.limiter.internalProcess("host", hostStr, ctx)
				return
			}
		}
		return
	}

	filter.limiter.internalProcess("host", hostStr, ctx)
}
