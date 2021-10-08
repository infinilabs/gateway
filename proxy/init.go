package proxy

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/filters/auth"
	"infini.sh/gateway/proxy/filters/cache"
	"infini.sh/gateway/proxy/filters/debug"
	elastic2 "infini.sh/gateway/proxy/filters/elastic"
	"infini.sh/gateway/proxy/filters/filter"
	"infini.sh/gateway/proxy/filters/ldap"
	"infini.sh/gateway/proxy/filters/rbac"
	"infini.sh/gateway/proxy/filters/routing"
	"infini.sh/gateway/proxy/filters/sample"
	"infini.sh/gateway/proxy/filters/throttle"
	"infini.sh/gateway/proxy/filters/transform"
	"infini.sh/gateway/proxy/output/elastic"
	"infini.sh/gateway/proxy/output/kafka"
	"infini.sh/gateway/proxy/output/logging"
	"infini.sh/gateway/proxy/output/queue"
	"infini.sh/gateway/proxy/output/redis_pubsub"
	queue2 "infini.sh/gateway/proxy/output/stats"
	"infini.sh/gateway/proxy/output/translog"
)

func Init() {
	common.RegisterFilterPlugin(logging.RequestLogging{})
	common.RegisterFilterPlugin(debug.EchoMessage{})
	common.RegisterFilterPlugin(debug.DumpHeader{})
	common.RegisterFilterPlugin(debug.DumpUrl{})
	common.RegisterFilterPlugin(debug.DumpRequestBody{})
	common.RegisterFilterPlugin(debug.DumpStatusCode{})
	common.RegisterFilterPlugin(debug.DumpResponseBody{})
	common.RegisterFilterPlugin(debug.DumpContext{})


	common.RegisterFilterPlugin(elastic.Elasticsearch{})
	common.RegisterFilterPlugin(elastic2.DatePrecisionTuning{})
	common.RegisterFilterPlugin(elastic2.BulkReshuffle{})
	common.RegisterFilterPlugin(elastic2.BulkToQueue{})
	common.RegisterFilterPlugin(elastic2.BulkResponseValidate{})

	common.RegisterFilterPlugin(cache.RequestCacheGet{})
	common.RegisterFilterPlugin(cache.RequestCacheSet{})

	common.RegisterFilterPlugin(throttle.RequestUserLimitFilter{})
	common.RegisterFilterPlugin(throttle.RequestHostLimitFilter{})
	common.RegisterFilterPlugin(throttle.RequestAPIKeyLimitFilter{})
	common.RegisterFilterPlugin(throttle.RequestClientIPLimitFilter{})
	common.RegisterFilterPlugin(throttle.RequestPathLimitFilter{})
	common.RegisterFilterPlugin(throttle.SleepFilter{})

	common.RegisterFilterPlugin(filter.RequestHeaderFilter{})
	common.RegisterFilterPlugin(filter.RequestMethodFilter{})
	common.RegisterFilterPlugin(sample.SampleFilter{})
	common.RegisterFilterPlugin(filter.RequestUrlPathFilter{})
	common.RegisterFilterPlugin(kafka.Kafka{})

	common.RegisterFilterPlugin(routing.RatioRoutingFlowFilter{})
	common.RegisterFilterPlugin(routing.CloneFlowFilter{})
	common.RegisterFilterPlugin(routing.SwitchFlowFilter{})
	common.RegisterFilterPlugin(routing.FlowFilter{})

	common.RegisterFilterPlugin(filter.ResponseStatusCodeFilter{})
	common.RegisterFilterPlugin(filter.ResponseHeaderFilter{})
	common.RegisterFilterPlugin(filter.RequestClientIPFilter{})
	common.RegisterFilterPlugin(filter.RequestUserFilter{})
	common.RegisterFilterPlugin(filter.RequestAPIKeyFilter{})
	common.RegisterFilterPlugin(filter.RequestServerHostFilter{})

	common.RegisterFilterPlugin(transform.RequestBodyTruncate{})
	common.RegisterFilterPlugin(transform.ResponseBodyTruncate{})
	common.RegisterFilterPlugin(transform.ResponseHeaderFormatFilter{})
	common.RegisterFilterPlugin(transform.RequestBodyRegexReplace{})
	common.RegisterFilterPlugin(transform.ResponseBodyRegexReplace{})

	common.RegisterFilterPlugin(transform.SetHostname{})
	common.RegisterFilterPlugin(transform.SetRequestHeader{})
	common.RegisterFilterPlugin(transform.SetRequestQueryArgs{})
	common.RegisterFilterPlugin(transform.SetResponseHeader{})
	common.RegisterFilterPlugin(transform.SetResponse{})
	common.RegisterFilterPlugin(transform.RequestBodyJsonSet{})
	common.RegisterFilterPlugin(transform.RequestBodyJsonDel{})

	common.RegisterFilterPlugin(auth.SetBasicAuth{})

	common.RegisterFilterPlugin(queue.DiskEnqueueFilter{})
	common.RegisterFilterPlugin(translog.TranslogOutput{})

	common.RegisterFilterPlugin(throttle.DropFilter{})
	common.RegisterFilterPlugin(throttle.ElasticsearchHealthCheckFilter{})

	common.RegisterFilterPlugin(redis_pubsub.RedisPubSub{})

	common.RegisterFilterPlugin(ldap.LDAPFilter{})
	common.RegisterFilterPlugin(rbac.RBACFilter{})


	common.RegisterFilterPlugin(throttle.RetryLimiter{})
	common.RegisterFilterPlugin(queue2.StatsFilter{})


}
