package throttle

import (
	"github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type ElasticsearchHealthCheckFilter struct {
	param.Parameters
}

func (filter ElasticsearchHealthCheckFilter) Name() string {
	return "elasticsearch_health_check"
}

func (filter ElasticsearchHealthCheckFilter) Process(ctx *fasthttp.RequestCtx) {
	esName := filter.MustGetString("elasticsearch")
	if rate.GetRateLimiter("cluster_check_health", esName, 1, 1, time.Second*1).Allow() {
		result := elastic.GetClient(esName).ClusterHealth()
		if global.Env().IsDebug {
			seelog.Trace(esName, result)
		}
		if result.StatusCode == 200 || result.StatusCode == 403 {
			cfg := elastic.GetMetadata(esName)
			if cfg != nil {
				cfg.ReportSuccess()
			}
		}
	}
}
