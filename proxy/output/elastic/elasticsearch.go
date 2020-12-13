package elastic

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/config"
	"sync"
)

type Elasticsearch struct {
	param.Parameters
}

func (filter Elasticsearch) Name() string {
	return "elasticsearch"
}

var proxyServer *ReverseProxy
var (
	proxyConfig = config.ProxyConfig{
		MaxConnection: 1000,
		//PassPatterns:   []string{"_cat", "scroll", "scroll_id", "_refresh", "_cluster", "_ccr", "_count", "_flush", "_ilm", "_ingest", "_license", "_migration", "_ml", "_nodes", "_rollup", "_data_stream", "_open", "_close"},
	}
)

//elasticsearch: default
//pass_pattern: ["_cat","scroll", "scroll_id","_refresh","_cluster","_ccr","_count","_flush","_ilm","_ingest","_license","_migration","_ml","_rollup","_data_stream","_open", "_close"]
//max_connection: 1000 #default for nodes
//timeout: 60s # default for nodes
//balancer: weight
//weight:
//- host: 192.168.3.1:9200
//weight: 10
//- host: 192.168.3.2:9200
//weight: 20
//discovery:
//enabled: false
//node_filter:
//- "coordinating"
//- "ingest"
//- "data"


var inited bool
var initLock sync.Mutex

func (filter Elasticsearch) Process(ctx *fasthttp.RequestCtx) {

	if !inited {
		initLock.Lock()
		defer initLock.Unlock()
		if inited{
			return
		}

		proxyConfig.Elasticsearch=filter.GetStringOrDefault("elasticsearch","default")
		proxyConfig.Balancer=filter.GetStringOrDefault("balancer","weight")
		proxyConfig.MaxResponseBodySize=filter.GetIntOrDefault("max_response_size",100 * 1024 * 1024)
		proxyConfig.MaxConnection=filter.GetIntOrDefault("max_connection",1000)
		filter.Config("discovery",&proxyConfig.Discover)
		proxyServer = NewReverseProxy(&proxyConfig)
		inited = true

	}

	proxyServer.DelegateRequest(&ctx.Request, &ctx.Response)

}
