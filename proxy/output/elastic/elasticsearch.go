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

package elastic

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type Elasticsearch struct {
	param.Parameters
	config   *ProxyConfig
	instance *ReverseProxy
	metadata *elastic.ElasticsearchMetadata
}

func (filter *Elasticsearch) Name() string {
	return "elasticsearch"
}

var faviconPath = []byte("/favicon.ico")

func (filter *Elasticsearch) Filter(ctx *fasthttp.RequestCtx) {

	if bytes.Equal(faviconPath, ctx.Request.Header.RequestURI()) {
		if global.Env().IsDebug {
			log.Tracef("skip to delegate favicon.io")
		}
		ctx.Finished()
		return
	}

	if !filter.config.SkipAvailableCheck && filter.getMetadata() != nil && !filter.getMetadata().IsAvailable() {
		if filter.config.CheckClusterHealthWhenNotAvailable {
			if rate.GetRateLimiter("cluster_check_health", filter.getMetadata().Config.ID, 1, 1, time.Second*1).Allow() {
				log.Debugf("Elasticsearch [%v] not available", filter.config.Elasticsearch)
				result, err := elastic.GetClient(filter.getMetadata().Config.Name).ClusterHealth(nil)
				if err != nil && result != nil && result.StatusCode == 200 {
					filter.getMetadata().ReportSuccess()
				}
			}
		}

		ctx.SetContentType(util.ContentTypeJson)
		ctx.Response.SwapBody([]byte(fmt.Sprintf("{\"error\":true,\"message\":\"Elasticsearch [%v] Service Unavailable\"}", filter.config.Elasticsearch)))
		ctx.SetStatusCode(503)
		ctx.Finished()
		return
	}

	//TODO move clients selection async
	filter.instance.DelegateRequest(filter.config.Elasticsearch, filter.getMetadata(), ctx)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("elasticsearch", New, &ProxyConfig{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	cfg := ProxyConfig{
		Balancer:                           "weight",
		MaxResponseBodySize:                100 * 1024 * 1024,
		MaxConnection:                      5000,
		MaxRetryTimes:                      5,
		RetryDelayInMs:                     1000,
		RetryReadonlyOnlyOnBackendFailure:  true,
		RetryOnBackendFailure:              true,
		TLSInsecureSkipVerify:              true,
		ReadBufferSize:                     4096 * 4,
		WriteBufferSize:                    4096 * 4,
		CheckClusterHealthWhenNotAvailable: true,
		//maxt wait timeout for free connection
		MaxConnWaitTimeout: util.GetDurationOrDefault("30s", 30*time.Second),

		//keep alived connection
		MaxConnDuration: util.GetDurationOrDefault("0s", 0*time.Second),

		Timeout:      util.GetDurationOrDefault("30s", 30*time.Second),
		DialTimeout:  util.GetDurationOrDefault("3s", 3*time.Second),
		ReadTimeout:  util.GetDurationOrDefault("0s", 0*time.Hour), //set to other value will cause broken error
		WriteTimeout: util.GetDurationOrDefault("0s", 0*time.Hour), //same as read timeout
		//idle alive connection will be closed
		MaxIdleConnDuration: util.GetDurationOrDefault("30s", 30*time.Second),
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := Elasticsearch{config: &cfg}
	runner.metadata = elastic.GetMetadata(cfg.Elasticsearch)

	runner.instance = NewReverseProxy(&cfg)

	log.Debugf("init elasticsearch proxy instance: %v", cfg.Elasticsearch)

	return &runner, nil
}

func (filter *Elasticsearch) getMetadata() *elastic.ElasticsearchMetadata {
	if filter.metadata == nil {
		filter.metadata = elastic.GetMetadata(filter.config.Elasticsearch)
	}
	return filter.metadata
}
