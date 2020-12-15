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

var proxyList=map[string]*ReverseProxy{}

var (
	proxyConfig = config.ProxyConfig{
		MaxConnection: 1000,
	}
)

var initLock sync.Mutex

func (filter Elasticsearch) Process(ctx *fasthttp.RequestCtx) {
	var instance *ReverseProxy

		esRef:=filter.GetStringOrDefault("elasticsearch","default")

		instance,ok:=proxyList[esRef]
		if !ok || instance==nil{
			initLock.Lock()
			defer initLock.Unlock()
			instance,ok=proxyList[esRef]
			//double check
			if !ok || instance==nil{

				proxyConfig.Elasticsearch=esRef
				proxyConfig.Balancer=filter.GetStringOrDefault("balancer","weight")
				proxyConfig.MaxResponseBodySize=filter.GetIntOrDefault("max_response_size",100 * 1024 * 1024)
				proxyConfig.MaxConnection=filter.GetIntOrDefault("max_connection",1000)

				if filter.Has("discovery"){
					filter.Config("discovery",&proxyConfig.Discover)
				}

				instance = NewReverseProxy(&proxyConfig)
				proxyList[esRef]=instance
			}
		}

	instance.DelegateRequest(&ctx.Request, &ctx.Response)

}
