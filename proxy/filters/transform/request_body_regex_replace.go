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

type RequestBodyRegexReplace struct {
	Pattern string `config:"pattern"`
	To      string `config:"to"`
	p       *regexp.Regexp
}

func (filter *RequestBodyRegexReplace) Name() string {
	return "request_body_regex_replace"
}

func (filter *RequestBodyRegexReplace) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("pattern:", filter.Pattern, ", to:", filter.To)
	}

	body := ctx.Request.GetRawBody()
	if len(body) > 0 {
		newBody := filter.p.ReplaceAll(body, util.UnsafeStringToBytes(filter.To))
		ctx.Request.SetRawBody(newBody)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_body_regex_replace",NewRequestBodyRegexReplace,&RequestBodyRegexReplace{})
}

func NewRequestBodyRegexReplace(c *config.Config) (filter pipeline.Filter, err error) {

	runner := RequestBodyRegexReplace{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.p, err = regexp.Compile(runner.Pattern)
	if err != nil {
		panic(err)
	}
	return &runner, nil
}
