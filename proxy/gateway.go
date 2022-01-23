package proxy

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
	api2 "infini.sh/gateway/api"
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/entry"
)

func ProxyHandler(ctx *fasthttp.RequestCtx) {

	stats.Increment("request", "total")

	//# Traffic Control Layer
	//Phase: eBPF based IP filter

	//Phase: XDP based traffic control, forward 1%-100% to another node, can be used for warming up or a/b testing

	//Phase: Handle Parameters, remove customized parameters and setup context

	//# DAG based Request Processing Flow
	//if reqFlowID!=""{
	//	flow.GetFlow(reqFlowID).Filter(ctx)
	//}
	//Phase: Requests Deny
	//TODO 根据请求IP和头信息,执行请求拒绝, 基于后台设置的黑白名单,执行准入, 只允许特定 IP Agent 访问 Gateway 访问

	//Phase: Deny Requests By Custom Rules, filter bad queries
	//TODO 慢查询,非法查询 主动检测和拒绝

	//Phase: Throttle Requests
	//Phase: Requests Decision
	//Phase: DAG based Filter
	//自动学习请求网站来生成 FST 路由信息, 基于 FST 数来快速路由

	//# Delegate Requests to upstream
	//proxyServer.DelegateRequest(&ctx.Request, &ctx.Response)

	//https://github.com/projectcontour/contour/blob/main/internal/dag/dag.go
	//Timeout Policy
	//Retry Policy
	//Virtual Policy
	//Routing Policy
	//Failback/Failsafe Policy

	//Phase: Handle Write Requests
	//Phase: Async Persist CUD

	//Phase: Cache Filter
	//TODO, no_cache -> skip cache and del query_args

	//Phase: Request Rewrite, reset @timestamp precision for Kibana

	//# Response Processing Flow
	//Phase: Recording

	//TODO 实时统计前后端 QPS, 出拓扑监控图
	//TODO 后台可以上传替换和编辑文件内容到缓存库里面, 直接返回自定义内容,如: favicon.ico, 可用于常用请求的提前预热,按 RequestURI 进行选择, 而不是完整 Hash

}

type GatewayModule struct {
	 api.Handler
	 entryConfigs []common.EntryConfig
	 entryPoints  map[string]*entry.Entrypoint
}

func (this *GatewayModule) Name() string {
	return "gateway"
}

func (module *GatewayModule) Setup(cfg *Config) {

	module.entryConfigs =[]common.EntryConfig{}
	module.entryPoints = map[string]*entry.Entrypoint{}

	initFilters()

	ok, err := env.ParseConfig("entry", &module.entryConfigs)
	if ok && err != nil {
		panic(err)
	}

	flowConfigs := []common.FlowConfig{}
	ok, err = env.ParseConfig("flow", &flowConfigs)
	if ok&&err != nil {
		panic(err)
	}
	if ok {
		for _, v := range flowConfigs {
			common.RegisterFlowConfig(v)
		}
	}

	routerConfigs := []common.RouterConfig{}
	ok, err = env.ParseConfig("router", &routerConfigs)
	if ok && err != nil {
		panic(err)
	}

	if ok {
		for _, v := range routerConfigs {
			common.RegisterRouterConfig(v)
		}
	}

	module.registerAPI("")

	api:= api2.GatewayAPI{}
	api.RegisterAPI("")
}


func (module *GatewayModule) Start() error {



	for _, v := range module.entryConfigs {
		entry := entry.NewEntrypoint(v)
		log.Trace("start entry:", entry.Name())
		err := entry.Start()
		if err != nil {
			panic(err)
		}
		module.entryPoints[v.Name] = entry
	}

	return nil
}

func (module *GatewayModule) Stop() error {

	for _, v := range module.entryPoints {
		err := v.Stop()
		if err != nil {
			panic(err)
		}
	}

	return nil
}
