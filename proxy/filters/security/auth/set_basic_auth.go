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

package auth

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type SetBasicAuth struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

func (filter *SetBasicAuth) Name() string {
	return "set_basic_auth"
}

func (filter *SetBasicAuth) Filter(ctx *fasthttp.RequestCtx) {

	//remove old one
	key, _ := ctx.Request.Header.PeekAnyKey(fasthttp.AuthHeaderKeys)
	if len(key) > 0 {
		ctx.Request.Header.Del(string(key))
	}

	//set new user
	ctx.Request.SetBasicAuth(filter.Username, filter.Password)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_basic_auth",NewSetBasicAuth,&SetBasicAuth{})
}

func NewSetBasicAuth(c *config.Config) (pipeline.Filter, error) {

	runner := SetBasicAuth{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
