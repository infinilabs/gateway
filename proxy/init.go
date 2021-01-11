package proxy

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/filter/cache"
	"infini.sh/gateway/proxy/filter/debug"
	elastic2 "infini.sh/gateway/proxy/filter/elastic"
	"infini.sh/gateway/proxy/filter/filters"
	"infini.sh/gateway/proxy/filter/routing"
	"infini.sh/gateway/proxy/filter/sample"
	"infini.sh/gateway/proxy/filter/throttle"
	"infini.sh/gateway/proxy/output/elastic"
	"infini.sh/gateway/proxy/output/kafka"
	"infini.sh/gateway/proxy/output/logging"
)

func Init() {
	common.RegisterFilter(logging.RequestLogging{})
	common.RegisterFilter(debug.EchoMessage{})
	common.RegisterFilter(debug.DumpHeader{})
	common.RegisterFilter(debug.DumpUrl{})
	common.RegisterFilter(elastic.Elasticsearch{})
	common.RegisterFilter(cache.RequestCacheGet{})
	common.RegisterFilter(cache.RequestCacheSet{})
	common.RegisterFilter(throttle.RateLimitFilter{})
	common.RegisterFilter(filters.RequestHeaderFilter{})
	common.RegisterFilter(filters.RequestMethodFilter{})
	common.RegisterFilter(sample.SampleFilter{})
	common.RegisterFilter(filters.RequestUrlPathFilter{})
	common.RegisterFilter(kafka.Kafka{})
	common.RegisterFilter(routing.RatioRoutingFlowFilter{})
	common.RegisterFilter(routing.CloneFlowFilter{})
	common.RegisterFilter(elastic2.BulkReshuffle{})
	common.RegisterFilter(debug.DumpRequestBody{})
	common.RegisterFilter(filters.RequestBodyTruncate{})
	common.RegisterFilter(filters.ResponseBodyTruncate{})
	common.RegisterFilter(filters.ResponseStatusCodeFilter{})
	common.RegisterFilter(filters.ResponseHeaderFilter{})
	common.RegisterFilter(filters.RequestClientIPFilter{})
	common.RegisterFilter(filters.RequestUserFilter{})
}
