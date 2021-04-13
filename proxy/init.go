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

	common.RegisterFilter(debug.DumpRequestBody{})
	common.RegisterFilter(debug.DumpResponseBody{})


	common.RegisterFilter(elastic.Elasticsearch{})
	common.RegisterFilter(elastic2.DatePrecisionTuning{})
	common.RegisterFilter(elastic2.BulkReshuffle{})
	common.RegisterFilter(elastic2.BulkToQueue{})

	common.RegisterFilter(cache.RequestCacheGet{})
	common.RegisterFilter(cache.RequestCacheSet{})

	common.RegisterFilter(throttle.RequestUserLimitFilter{})
	common.RegisterFilter(throttle.RequestHostLimitFilter{})
	common.RegisterFilter(throttle.RequestAPIKeyLimitFilter{})
	common.RegisterFilter(throttle.RequestClientIPLimitFilter{})
	common.RegisterFilter(throttle.RequestPathLimitFilter{})
	common.RegisterFilter(throttle.SleepFilter{})

	common.RegisterFilter(filter.RequestHeaderFilter{})
	common.RegisterFilter(filter.RequestMethodFilter{})
	common.RegisterFilter(sample.SampleFilter{})
	common.RegisterFilter(filter.RequestUrlPathFilter{})
	common.RegisterFilter(kafka.Kafka{})

	common.RegisterFilter(routing.RatioRoutingFlowFilter{})
	common.RegisterFilter(routing.CloneFlowFilter{})
	common.RegisterFilter(routing.SwitchFlowFilter{})

	common.RegisterFilter(filter.ResponseStatusCodeFilter{})
	common.RegisterFilter(filter.ResponseHeaderFilter{})
	common.RegisterFilter(filter.RequestClientIPFilter{})
	common.RegisterFilter(filter.RequestUserFilter{})
	common.RegisterFilter(filter.RequestAPIKeyFilter{})
	common.RegisterFilter(filter.RequestServerHostFilter{})

	common.RegisterFilter(transform.RequestBodyTruncate{})
	common.RegisterFilter(transform.ResponseBodyTruncate{})
	common.RegisterFilter(transform.ResponseHeaderFormatFilter{})
	common.RegisterFilter(transform.RequestBodyRegexReplace{})
	common.RegisterFilter(transform.ResponseBodyRegexReplace{})
}
