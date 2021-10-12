package proxy

import (
	"infini.sh/framework/core/pipeline"
	"infini.sh/gateway/proxy/filters/cache"
	"infini.sh/gateway/proxy/filters/debug/dump"
	"infini.sh/gateway/proxy/filters/debug/echo"
	elastic2 "infini.sh/gateway/proxy/filters/elastic"
	"infini.sh/gateway/proxy/filters/elastic/date_range_precision_tuning"
	"infini.sh/gateway/proxy/filters/filter"
	"infini.sh/gateway/proxy/filters/routing"
	"infini.sh/gateway/proxy/filters/sample"
	"infini.sh/gateway/proxy/filters/security/auth"
	"infini.sh/gateway/proxy/filters/security/ldap"
	"infini.sh/gateway/proxy/filters/security/rbac"
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
	pipeline.RegisterFilterPlugin("echo", echo.New)
	pipeline.RegisterFilterPlugin("logging", logging.New)
	pipeline.RegisterFilterPlugin("elasticsearch", elastic.New)
	pipeline.RegisterFilterPlugin("get_cache", cache.NewGet)
	pipeline.RegisterFilterPlugin("set_cache", cache.NewSet)
	pipeline.RegisterFilterPlugin("dump", dump.New)
	pipeline.RegisterFilterPlugin("date_range_precision_tuning", date_range_precision_tuning.New)
	pipeline.RegisterFilterPlugin("bulk_reshuffle", pipeline.FilterConfigChecked(elastic2.NewBulkReshuffle, pipeline.RequireFields("elasticsearch")))
	pipeline.RegisterFilterPlugin("bulk_response_validate", elastic2.NewBulkResponseValidate)
	pipeline.RegisterFilterPlugin("drop", throttle.NewDropFilter)
	pipeline.RegisterFilterPlugin("elasticsearch_health_check", throttle.NewHealthCheckFilter)
	pipeline.RegisterFilterPlugin("sleep",throttle.NewSleepFilter)
	pipeline.RegisterFilterPlugin("retry_limiter",pipeline.FilterConfigChecked(throttle.NewRetryLimiter, pipeline.RequireFields("queue_name")))
	pipeline.RegisterFilterPlugin("request_user_limiter",throttle.NewRequestUserLimitFilter)
	pipeline.RegisterFilterPlugin("request_host_limiter",throttle.NewRequestHostLimitFilter)
	pipeline.RegisterFilterPlugin("request_api_key_limiter",throttle.NewRequestAPIKeyLimitFilter)
	pipeline.RegisterFilterPlugin("request_client_ip_limiter",throttle.NewRequestClientIPLimitFilter)
	pipeline.RegisterFilterPlugin("request_path_limiter",throttle.NewRequestPathLimitFilter)
	pipeline.RegisterFilterPlugin("sample",sample.NewSampleFilter)
	pipeline.RegisterFilterPlugin("request_body_regex_replace",transform.NewRequestBodyRegexReplace)
	pipeline.RegisterFilterPlugin("response_body_regex_replace",transform.NewResponseBodyRegexReplace)
	pipeline.RegisterFilterPlugin("request_body_json_del",transform.NewRequestBodyJsonDel)
	pipeline.RegisterFilterPlugin("request_body_json_set",transform.NewRequestBodyJsonSet)
	pipeline.RegisterFilterPlugin("ratio",routing.NewRatioRoutingFlowFilter)
	pipeline.RegisterFilterPlugin("clone",routing.NewCloneFlowFilter)
	pipeline.RegisterFilterPlugin("switch",routing.NewSwitchFlowFilter)
	pipeline.RegisterFilterPlugin("flow",routing.NewFlowFilter)

	pipeline.RegisterFilterPlugin("request_method_filter",filter.NewRequestMethodFilter)
	pipeline.RegisterFilterPlugin("request_path_filter",filter.NewRequestUrlPathFilter)
	pipeline.RegisterFilterPlugin("request_header_filter",filter.NewRequestHeaderFilter)
	pipeline.RegisterFilterPlugin("request_client_ip_filter",filter.NewRequestClientIPFilter)
	pipeline.RegisterFilterPlugin("request_user_filter",filter.NewRequestUserFilter)
	pipeline.RegisterFilterPlugin("request_api_key_filter",filter.NewRequestAPIKeyFilter)
	pipeline.RegisterFilterPlugin("request_host_filter",filter.NewRequestServerHostFilter)

	pipeline.RegisterFilterPlugin("response_header_filter",filter.NewResponseHeaderFilter)
	pipeline.RegisterFilterPlugin("response_status_filter",filter.NewResponseStatusCodeFilter)



	//废弃的过滤器
	//pipeline.RegisterFilterPlugin(transform.RequestBodyTruncate{})
	//pipeline.RegisterFilterPlugin(transform.ResponseBodyTruncate{})

	pipeline.RegisterFilterPlugin("response_header_format",transform.NewResponseHeaderFormatFilter)

	pipeline.RegisterFilterPlugin("set_hostname",transform.NewSetHostname)
	pipeline.RegisterFilterPlugin("set_request_header",transform.NewSetRequestHeader)
	pipeline.RegisterFilterPlugin("set_request_query_args",transform.NewSetRequestQueryArgs)
	pipeline.RegisterFilterPlugin("set_response_header",transform.NewSetResponseHeader)
	pipeline.RegisterFilterPlugin("set_response",transform.NewSetResponse)
	pipeline.RegisterFilterPlugin("set_basic_auth",auth.NewSetBasicAuth)

	pipeline.RegisterFilterPlugin("queue",queue.NewDiskEnqueueFilter)
	pipeline.RegisterFilterPlugin("translog",translog.NewTranslogOutput)
	pipeline.RegisterFilterPlugin("redis_pubsub",pipeline.FilterConfigChecked(redis_pubsub.NewRedisPubSub, pipeline.RequireFields("channel")))
	pipeline.RegisterFilterPlugin("kafka",kafka.NewKafkaFilter)


	pipeline.RegisterFilterPlugin("ldap_auth",pipeline.FilterConfigChecked(ldap.NewLDAPFilter, pipeline.RequireFields("host","bind_dn","base_dn")))
	pipeline.RegisterFilterPlugin("rbac",rbac.NewRBACFilter)

	pipeline.RegisterFilterPlugin("stats",queue2.NewStatsFilter)

}
