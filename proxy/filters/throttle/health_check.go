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

package throttle

import (
	"fmt"
	"github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type ElasticsearchHealthCheckFilter struct {
	Elasticsearch string `config:"elasticsearch"`
	Interval      int    `config:"interval"`
}

func (filter *ElasticsearchHealthCheckFilter) Name() string {
	return "elasticsearch_health_check"
}

func (filter *ElasticsearchHealthCheckFilter) Filter(ctx *fasthttp.RequestCtx) {
	if rate.GetRateLimiter("cluster_check_health", filter.Elasticsearch, 1, 1, time.Second*time.Duration(filter.Interval)).Allow() {
		result, _ := elastic.GetClient(filter.Elasticsearch).ClusterHealth(nil)
		if global.Env().IsDebug {
			seelog.Trace(filter.Elasticsearch, result)
		}
		if result.StatusCode == 200 || result.StatusCode == 403 {
			cfg := elastic.GetMetadata(filter.Elasticsearch)
			if cfg != nil {
				cfg.ReportSuccess()
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("elasticsearch_health_check", NewHealthCheckFilter, &ElasticsearchHealthCheckFilter{})
}

func NewHealthCheckFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchHealthCheckFilter{
		Interval: 1,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
