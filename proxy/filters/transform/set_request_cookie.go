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

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package transform

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type Cookie struct {
	Reset bool `config:"reset"`//reset request cookies
	Cookies map[string]string `config:"cookies"`//request cookies
}

func (filter *Cookie) Name() string {
	return "set_request_cookie"
}

func (filter *Cookie) Filter(ctx *fasthttp.RequestCtx) {
	if filter.Reset{
		ctx.Request.Header.DelAllCookies()
	}

	for k,v:=range filter.Cookies{
		ctx.Request.Header.SetCookie(k,v)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_request_cookie", NewCookieFilter,&Cookie{})
}

func NewCookieFilter(c *config.Config) (pipeline.Filter, error) {

	runner := Cookie{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
