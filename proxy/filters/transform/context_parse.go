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

package transform

import (
	"fmt"
	"regexp"

	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ContextParseFilter struct {
	SkipError bool   `config:"skip_error"`
	Context   string `config:"context"`
	Pattern   string `config:"pattern"`
	Group     string `config:"group"`
	p         *regexp.Regexp
}

func (filter *ContextParseFilter) Name() string {
	return "context_parse"
}

func (filter *ContextParseFilter) Filter(ctx *fasthttp.RequestCtx) {
	if filter.Context != "" {
		key, err := ctx.GetValue(filter.Context)
		if err != nil {
			if filter.SkipError {
				return
			}
			panic(errors.Errorf("context_parse,url:%v,err:%v", ctx.Request.PhantomURI().String(), err))
		}
		keyStr := util.ToString(key)
		variables := util.MapStr{}
		if filter.p != nil {
			match := filter.p.FindStringSubmatch(keyStr)
			if len(match) > 0 {
				for i, name := range filter.p.SubexpNames() {
					if name != "" {
						variables[name] = match[i]
					}
				}
			}
		}
		if len(variables) > 0 {
			if filter.Group != "" {
				_, err = ctx.PutValue(filter.Group, variables)
				if err != nil {
					if filter.SkipError {
						return
					}
					panic(err)
				}
			} else {
				for k, v := range variables {
					_, err = ctx.PutValue(k, v)
					if err != nil {
						if filter.SkipError {
							return
						}
						panic(err)
					}
				}
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_parse", NewContextParseFilter, &ContextParseFilter{})
}

func NewContextParseFilter(c *config.Config) (pipeline.Filter, error) {
	runner := ContextParseFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	var err error
	if runner.Context != "" && runner.Pattern != "" {
		runner.p, err = regexp.Compile(runner.Pattern)
		if err != nil {
			panic(err)
		}
	}

	return &runner, nil
}
