package proxy

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/fasthttp"
	api2 "infini.sh/gateway/api"
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/entry"
	"runtime"
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

	entryPoints map[string]*entry.Entrypoint

	API struct {
		Enabled bool `config:"enabled"`
	} `config:"api"`

	ORM struct {
		Enabled bool `config:"enabled"`
	} `config:"orm"`

	DisableReusePortByDefault bool `config:"disable_reuse_port_by_default"`
}

func (this *GatewayModule) Name() string {
	return "gateway"
}

func (module *GatewayModule) Setup(cfg *Config) {

	module.entryPoints = module.loadEntryPoints()

	api := api2.GatewayAPI{}
	if module.API.Enabled {
		api.RegisterAPI("")
	}
	if module.ORM.Enabled {
		api.RegisterSchema()
	}

	module.registerAPI("")

	module.handleConfigureChange()
}

func (module *GatewayModule) handleConfigureChange(){

	NotifyOnConfigSectionChange("flow", func(pCfg,cCfg *Config) {

		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error on apply flow change,", v)
				}
			}
		}()

		if cCfg!=nil{
			//TODO diff previous and current config
			newConfig:= []common.FlowConfig{}
			err:=cCfg.Unpack(&newConfig)
			if err!=nil{
				log.Error(err)
				return
			}

			for _, v := range newConfig {
				common.RegisterFlowConfig(v)
			}

			////just in case
			//for _,v:=range module.entryPoints{
			//	v.RefreshTracingFlow()
			//}
		}
	})

	NotifyOnConfigSectionChange("router", func(pCfg,cCfg *Config) {
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error on apply router change,", v)
				}
			}
		}()

		if cCfg!=nil{
			newConfig:= []common.RouterConfig{}
			err:=cCfg.Unpack(&newConfig)
			if err!=nil{
				log.Error(err)
				return
			}

			keys:=map[string]string{}
			for _, v := range newConfig {
				keys[v.ID]=v.ID
				common.RegisterRouterConfig(v)
			}

			//修改完路由，需要重启服务入口
			for _,v:=range module.entryPoints{
				_,ok:=keys[v.GetConfig().RouterConfigName]
				if ok{
					v.Stop()
					v.Start()
				}
			}
		}
	})

	NotifyOnConfigSectionChange("entry", func(pCfg,cCfg *Config) {

		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error on apply entry change,", v)
				}
			}
		}()

		if cCfg!=nil{
			newConfig:=[]common.EntryConfig{}
			err:=cCfg.Unpack(&newConfig)
			if err!=nil{
				log.Error(err)
				return
			}

			//each entry should reuse port
			old:=module.entryPoints
			skipKeys:=map[string]string{}
			entryPoints := map[string]*entry.Entrypoint{}
			for _, v := range newConfig {
				oldC,ok:=old[v.ID]
				if ok{
					config := oldC.GetConfig()
					if config.Equals(&v){
						skipKeys[v.RouterConfigName]=v.RouterConfigName
						continue
					}
				}

				if !module.DisableReusePortByDefault{
					v.NetworkConfig.ReusePort=true
				}
				e := entry.NewEntrypoint(v)
				entryPoints[v.ID] = e
			}

			if len(entryPoints)==0{
				return
			}

			log.Debug("starting new entry points")
			for _,v:=range entryPoints{
				v.Start()
			}

			log.Debug("stopping old entry points")
			for _,v:=range old{
				_,ok:=skipKeys[v.GetConfig().ID]
				if ok{
					entryPoints[v.GetConfig().ID]=v
					continue
				}
				v.Stop()
			}

			module.entryPoints=entryPoints
		}
	})

}

func (module *GatewayModule) loadEntryPoints()map[string]*entry.Entrypoint {

	routerConfigs := []common.RouterConfig{}
	flowConfigs := []common.FlowConfig{}
	entryConfigs := []common.EntryConfig{}

	ok, err := env.ParseConfig("gateway", &module)
	if ok && err != nil {
		panic(err)
	}

	ok, err = env.ParseConfig("entry", &entryConfigs)
	if ok && err != nil {
		panic(err)
	}

	ok, err = env.ParseConfig("flow", &flowConfigs)
	if ok && err != nil {
		panic(err)
	}

	if ok {
		for _, v := range flowConfigs {
			common.RegisterFlowConfig(v)
		}
	}

	ok, err = env.ParseConfig("router", &routerConfigs)
	if ok && err != nil {
		panic(err)
	}

	if ok {
		for _, v := range routerConfigs {
			common.RegisterRouterConfig(v)
		}
	}

	entryPoints := map[string]*entry.Entrypoint{}
	for _, v := range entryConfigs {
		if !module.DisableReusePortByDefault{
			v.NetworkConfig.ReusePort=true
		}
		e := entry.NewEntrypoint(v)
		entryPoints[v.ID] = e
	}
	return entryPoints
}


func (module *GatewayModule) Start() error {

	for _, v := range module.entryPoints {
		log.Trace("start entry:", v.String())
		err := v.Start()
		if err != nil {
			panic(err)
		}
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
