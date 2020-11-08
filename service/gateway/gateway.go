package gateway

import (
	log "github.com/cihub/seelog"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/api"
	"infini.sh/gateway/common"
	"infini.sh/gateway/config"
	"infini.sh/gateway/proxy/filter"
	"infini.sh/gateway/proxy/output/translog"
	proxy "infini.sh/gateway/proxy/reverse-proxy"
)

func ProxyHandler(ctx *fasthttp.RequestCtx) {

	stats.Increment("request", "total")

	//# Traffic Control Layer
	//Phase: eBPF based IP filter

	//Phase: XDP based traffic control, forward 1%-100% to another node, can be used for warming up or a/b testing

	//# Route Layer
	//router:= router.NewRouter()
	//reqFlowID,resFlowID:=router.GetFlow(ctx)

	//Phase: Handle Parameters, remove customized parameters and setup context

	//# DAG based Request Processing Flow
	//if reqFlowID!=""{
	//	flow.GetFlow(reqFlowID).Process(ctx)
	//}
	//Phase: Requests Deny
	//TODO 根据请求IP和头信息,执行请求拒绝, 基于后台设置的黑白名单,执行准入, 只允许特定 IP Agent 访问 Gateway 访问

	//Phase: Deny Requests By Custom Rules, filter bad queries
	//TODO 慢查询,非法查询 主动检测和拒绝

	//Phase: Throttle Requests

	//Phase: Requests Decision
	//Phase: DAG based Process
	//自动学习请求网站来生成 FST 路由信息, 基于 FST 数来快速路由

	//# Delegate Requests to upstream
	proxyServer.DelegateRequest(&ctx.Request, &ctx.Response)

	//https://github.com/projectcontour/contour/blob/main/internal/dag/dag.go
	//Timeout Policy
	//Retry Policy
	//Virtual Policy
	//Routing Policy
	//Failback/Failsafe Policy

	//Phase: Handle Write Requests
	//Phase: Async Persist CUD

	//Phase: Cache Process
	//TODO, no_cache -> skip cache and del query_args

	//Phase: Request Rewrite, reset @timestamp precision for Kibana

	//# Response Processing Flow
	//if resFlowID!=""{
	//	flow.GetFlow(resFlowID).Process(ctx)
	//}
	//Phase: Recording
	//TODO 记录所有请求,采样记录,按条件记录

	//TODO 实时统计前后端 QPS, 出拓扑监控图
	//TODO 后台可以上传替换和编辑文件内容到缓存库里面, 直接返回自定义内容,如: favicon.ico, 可用于常用请求的提前预热,按 RequestURI 进行选择, 而不是完整 Hash

	//logging event
	//TODO configure log req and response, by condition
}

type GatewayModule struct {
}

func (this GatewayModule) Name() string {
	return "gateway"
}

var (
	proxyConfig = config.ProxyConfig{
		MaxConcurrency:      1000,
		PassthroughPatterns: []string{"_cat", "scroll", "scroll_id", "_refresh", "_cluster", "_ccr", "_count", "_flush", "_ilm", "_ingest", "_license", "_migration", "_ml", "_nodes", "_rollup", "_data_stream", "_open", "_close"},
	}
)
var proxyServer *proxy.ReverseProxy

//
////var proxyServer *proxy.ReverseProxy
//
//func LoadConfig() {
//
//	env.ParseConfig("proxy", &proxyConfig)
//
//	if !proxyConfig.Enabled {
//		return
//	}
//
//	config.SetProxyConfig(proxyConfig)
//
//	api.Init()
//	filter.Init()
//
//	proxyServer = proxy.NewReverseProxy(&proxyConfig)
//
//	//init router, and default handler to
//	router = r.New()
//	router.NotFound = proxyServer.DelegateToUpstream
//
//	if global.Env().IsDebug{
//		log.Trace("tracing enabled:", proxyConfig.TracingEnabled)
//	}
//
//	if proxyConfig.TracingEnabled {
//		router.OnFinishHandler = common.GetFlowProcess("request_logging")
//	}
//
//}

func (module GatewayModule) Setup(cfg *Config) {

	env.ParseConfig("proxy", &proxyConfig)

	if !proxyConfig.Enabled {
		return
	}

	config.SetProxyConfig(proxyConfig)

	api.Init()
	filter.Init()

	proxyServer = proxy.NewReverseProxy(&proxyConfig)

	//init router, and default handler to
	router.NotFound = proxyServer.DelegateToUpstream

	if global.Env().IsDebug{
		log.Trace("tracing enabled:", proxyConfig.TracingEnabled)
	}

	if proxyConfig.TracingEnabled {
		router.OnFinishHandler = common.GetFlowProcess("request_logging")
	}

}

func (module GatewayModule) Start() error {

	if !proxyConfig.Enabled {
		return nil
	}

	translog.Open()

	return nil
}

func (module GatewayModule) Stop() error {
	if !proxyConfig.Enabled {
		return nil
	}

	translog.Close()

	return nil
}
