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

type RequestUserLimitFilter struct {
	limiter *GenericLimiter
	User    []string `config:"user"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_user_limiter", NewRequestUserLimitFilter, &RequestUserLimitFilter{})
}

func NewRequestUserLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestUserLimitFilter{}

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

func (filter *RequestUserLimitFilter) Name() string {
	return "request_user_limiter"
}

func (filter *RequestUserLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	exists, user, _ := ctx.Request.ParseBasicAuth()
	if !exists {
		if global.Env().IsDebug {
			log.Tracef("user not exist")
		}
		return
	}

	userStr := string(user)
	if global.Env().IsDebug {
		log.Trace("user rules: ", len(filter.User), ", user: ", userStr)
	}

	if len(filter.User) > 0 {
		for _, v := range filter.User {
			if v == userStr {
				if global.Env().IsDebug {
					log.Debug(userStr, "met check rules")
				}
				filter.limiter.internalProcess("user", userStr, ctx)
				return
			}
		}
		//terminate if no ip matches
		return
	}

	filter.limiter.internalProcess("user", userStr, ctx)
}
