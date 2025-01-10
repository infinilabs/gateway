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

package transform

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type ContextRegexReplace struct {
	Context string `config:"context"`
	Pattern string `config:"pattern"`
	To      string `config:"to"`
	p       *regexp.Regexp
}

func (filter *ContextRegexReplace) Name() string {
	return "context_regex_replace"
}

func (filter *ContextRegexReplace) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("context:", filter.Context, "pattern:", filter.Pattern, ", to:", filter.To)
	}

	if filter.Context != "" {
		value, err := ctx.GetValue(filter.Context)
		if err != nil {
			log.Error(err)
			return
		}
		valueStr := util.ToString(value)
		if len(valueStr) > 0 {
			newBody := filter.p.ReplaceAll([]byte(valueStr), util.UnsafeStringToBytes(filter.To))
			_, err := ctx.PutValue(filter.Context, string(newBody))
			if err != nil {
				log.Error(err)
				return
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_regex_replace", NewContextRegexReplace, &ContextRegexReplace{})
}

func NewContextRegexReplace(c *config.Config) (filter pipeline.Filter, err error) {

	runner := ContextRegexReplace{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.p, err = regexp.Compile(runner.Pattern)
	if err != nil {
		panic(err)
	}
	return &runner, nil
}
