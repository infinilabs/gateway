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

package queue

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
)

type StatsFilter struct {
	Category string `config:"category"`
}

func (filter StatsFilter) Name() string {
	return "stats"
}

func (filter StatsFilter) Filter(ctx *fasthttp.RequestCtx) {

	stats.Timing(filter.Category, "response.elapsed_ms", ctx.GetElapsedTime().Milliseconds())
	stats.IncrementBy(filter.Category, "response.bytes", int64(ctx.Response.GetResponseLength()))
	stats.Increment(filter.Category, fmt.Sprintf("response.status.%v", ctx.Response.StatusCode()))

	stats.IncrementBy(filter.Category, "request.bytes", int64(ctx.Request.GetRequestLength()))
	stats.Increment(filter.Category, fmt.Sprintf("request.method.%v", string(ctx.Request.Header.Method())))

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("stats",NewStatsFilter,&StatsFilter{})
}

func NewStatsFilter(c *config.Config) (pipeline.Filter, error) {

	runner := StatsFilter{
		Category: global.Env().GetAppLowercaseName(),
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
