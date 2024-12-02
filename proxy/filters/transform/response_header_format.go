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

	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseHeaderFormatFilter struct {
}

func (filter *ResponseHeaderFormatFilter) Name() string {
	return "response_header_format"
}

func (filter *ResponseHeaderFormatFilter) Filter(ctx *fasthttp.RequestCtx) {

	ctx.Response.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.SetBytesKV(util.ToLowercase(key), value)
	})
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("response_header_format", NewResponseHeaderFormatFilter, &ResponseHeaderFormatFilter{})
}

func NewResponseHeaderFormatFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ResponseHeaderFormatFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
