package elastic

import (
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/config"
	proxy "infini.sh/gateway/proxy/reverse-proxy"
)

type Elasticsearch struct {
	pipeline.Parameters
}

func (filter Elasticsearch) Name() string {
	return "elasticsearch"
}

var proxyServer *proxy.ReverseProxy
var (
	proxyConfig = config.ProxyConfig{
		MaxConcurrency:      1000,
		PassthroughPatterns: []string{"_cat", "scroll", "scroll_id", "_refresh", "_cluster", "_ccr", "_count", "_flush", "_ilm", "_ingest", "_license", "_migration", "_ml", "_nodes", "_rollup", "_data_stream", "_open", "_close"},
	}
)
var inited bool
var direct bool

func (filter Elasticsearch) Process(ctx *fasthttp.RequestCtx) {

	if !inited {
		ok, err := env.ParseConfig("proxy", &proxyConfig)
		if err != nil {
			panic(err)
		}
		if ok {
			config.SetProxyConfig(proxyConfig)
			proxyServer = proxy.NewReverseProxy(&proxyConfig)
		}
		inited = true
		direct = filter.GetBool("direct", true)
	}

	if direct {
		proxyServer.DelegateRequest(&ctx.Request, &ctx.Response)
		return
	}

	//size:=joint.GetIntOrDefault("repeat",1)
	proxyServer.DelegateToUpstream(ctx)

}
