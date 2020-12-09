package proxy

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/filter/cache"
	"infini.sh/gateway/proxy/filter/filters"
	"infini.sh/gateway/proxy/filter/throttle"
	"infini.sh/gateway/proxy/output/elastic"
	"infini.sh/gateway/proxy/output/logging"
	"infini.sh/gateway/proxy/output/echo"
)

func Init()  {
	common.RegisterFilter(logging.RequestLogging{})
	common.RegisterFilter(echo.EchoDot{})
	common.RegisterFilter(elastic.Elasticsearch{})
	common.RegisterFilter(cache.RequestCacheGet{})
	common.RegisterFilter(cache.RequestCacheSet{})
	common.RegisterFilter(throttle.RateLimitFilter{})
	common.RegisterFilter(filters.RequestHeaderFilter{})
}
