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

type RequestAPIKeyLimitFilter struct {
	limiter *GenericLimiter
	APIKeys []string `config:"id"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_api_key_limiter", NewRequestAPIKeyLimitFilter, &RequestAPIKeyLimitFilter{})
}

func NewRequestAPIKeyLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestAPIKeyLimitFilter{}
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

func (filter *RequestAPIKeyLimitFilter) Name() string {
	return "request_api_key_limiter"
}

func (filter *RequestAPIKeyLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	exists, apiID, _ := ctx.ParseAPIKey()
	if !exists {
		if global.Env().IsDebug {
			log.Tracef("api not exist")
		}
		return
	}

	apiIDStr := string(apiID)
	if global.Env().IsDebug {
		log.Trace("api rules: ", len(filter.APIKeys), ", api: ", apiIDStr)
	}

	if len(filter.APIKeys) > 0 {
		for _, v := range filter.APIKeys {
			if v == apiIDStr {
				if global.Env().IsDebug {
					log.Debug(apiIDStr, "met check rules")
				}
				filter.limiter.internalProcess("api", apiIDStr, ctx)
				return
			}
		}
		return
	}

	filter.limiter.internalProcess("api", apiIDStr, ctx)
}
