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

/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package echo

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type Tag struct {
	AddTags       []string `config:"add" `
	RemoveTags    []string `config:"remove" `
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("tag", New, &Tag{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	runner := Tag{
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *Tag) Name() string {
	return "tag"
}


func (filter *Tag) Filter(ctx *fasthttp.RequestCtx) {

		if len(filter.AddTags)>0||len(filter.RemoveTags)>0{
			ctx.UpdateTags(filter.AddTags,filter.RemoveTags)
		}

}
