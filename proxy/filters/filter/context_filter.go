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

type ContextFilter struct {
	genericFilter *RequestFilter
	Context       string   `config:"context"`
	Include       []string `config:"include"`
	Exclude       []string `config:"exclude"`
}

func (filter *ContextFilter) Name() string {
	return "context_filter"
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_filter", NewContextFilter, &ContextFilter{})
}

func NewContextFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ContextFilter{}
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

func (filter *ContextFilter) Filter(ctx *fasthttp.RequestCtx) {
	context, err := ctx.GetValue(filter.Context)
	if err != nil {
		log.Debugf("context [%v] not exist", context)
		return
	}

	str := util.ToString(context)

	if len(filter.Exclude) > 0 {
		valid, hasRule := CheckExcludeStringRules(str, filter.Exclude, ctx)
		if hasRule && !valid {
			filter.genericFilter.Filter(ctx)
			if global.Env().IsDebug {
				log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
			}
			return
		}
	}

	if len(filter.Include) > 0 {
		valid, hasRule := CheckIncludeStringRules(str, filter.Include, ctx)
		if hasRule && !valid {
			filter.genericFilter.Filter(ctx)
			if global.Env().IsDebug {
				log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
			}
			return
		}
	}

	var hasOtherRules = false
	var hasRules = false
	var valid = false

	valid, hasRules = filter.genericFilter.CheckMustNotRules(str, ctx)
	if !valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	valid, hasRules = filter.genericFilter.CheckMustRules(str, ctx)

	if !valid {
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("must rules not matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	var hasShouldRules = false
	valid, hasShouldRules = filter.genericFilter.CheckShouldRules(str, ctx)
	if !valid {
		if !hasOtherRules && hasShouldRules {
			filter.genericFilter.Filter(ctx)
			if global.Env().IsDebug {
				log.Debugf("only should rules, but none of them are matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
			}
		}
	}

}
