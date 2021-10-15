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
	Interval int `config:"interval"`
}

func (filter *ElasticsearchHealthCheckFilter) Name() string {
	return "elasticsearch_health_check"
}

func (filter *ElasticsearchHealthCheckFilter) Filter(ctx *fasthttp.RequestCtx) {
	if rate.GetRateLimiter("cluster_check_health", filter.Elasticsearch, 1, 1, time.Second*time.Duration(filter.Interval)).Allow() {
		result := elastic.GetClient(filter.Elasticsearch).ClusterHealth()
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

func NewHealthCheckFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchHealthCheckFilter{
		Interval: 1,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
