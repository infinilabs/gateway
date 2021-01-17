package proxy

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/filters/cache"
	"infini.sh/gateway/proxy/filters/debug"
	elastic2 "infini.sh/gateway/proxy/filters/elastic"
	"infini.sh/gateway/proxy/filters/filter"
	"infini.sh/gateway/proxy/filters/routing"
	"infini.sh/gateway/proxy/filters/sample"
	"infini.sh/gateway/proxy/filters/throttle"
	"infini.sh/gateway/proxy/filters/transform"
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
	common.RegisterFilter(filter.RequestHeaderFilter{})
	common.RegisterFilter(filter.RequestMethodFilter{})
	common.RegisterFilter(sample.SampleFilter{})
	common.RegisterFilter(filter.RequestUrlPathFilter{})
	common.RegisterFilter(kafka.Kafka{})
	common.RegisterFilter(routing.RatioRoutingFlowFilter{})
	common.RegisterFilter(routing.CloneFlowFilter{})
	common.RegisterFilter(elastic2.BulkReshuffle{})
	common.RegisterFilter(debug.DumpRequestBody{})
	common.RegisterFilter(transform.RequestBodyTruncate{})
	common.RegisterFilter(transform.ResponseBodyTruncate{})
	common.RegisterFilter(filter.ResponseStatusCodeFilter{})
	common.RegisterFilter(filter.ResponseHeaderFilter{})
	common.RegisterFilter(filter.RequestClientIPFilter{})
	common.RegisterFilter(filter.RequestUserFilter{})
	common.RegisterFilter(filter.RequestServerHostFilter{})
	common.RegisterFilter(elastic2.DatePrecisionTuning{})
}
