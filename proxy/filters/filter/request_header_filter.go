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

package filter

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type RequestHeaderFilter struct {
	genericFilter *RequestFilter

	Include []map[string]string `config:"include"`
	Exclude []map[string]string `config:"exclude"`
}

func (filter *RequestHeaderFilter) Name() string {
	return "request_header_filter"
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_header_filter", NewRequestHeaderFilter, &RequestHeaderFilter{})
}

func NewRequestHeaderFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestHeaderFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.genericFilter = &RequestFilter{
		Action: "deny",
		Status: 403,
	}

	if err := c.Unpack(runner.genericFilter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter RequestHeaderFilter) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Debug("headers:", util.UnsafeBytesToString(util.EscapeNewLine(ctx.Request.Header.Header())))
	}

	if len(filter.Exclude) > 0 {
		for _, x := range filter.Exclude {
			for k, v := range x {
				v1 := ctx.Request.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("exclude header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					filter.genericFilter.Filter(ctx)
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
					}
					return
				}
			}
		}
	}

	if len(filter.Include) > 0 {
		for _, x := range filter.Include {
			for k, v := range x {
				v1 := ctx.Request.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("include header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.PhantomURI().String())
					}
					return
				}
			}
		}
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
		}
	}

}
