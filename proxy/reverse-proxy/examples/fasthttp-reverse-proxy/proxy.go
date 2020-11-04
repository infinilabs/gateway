package main

import (
	"strings"

	"github.com/yeqown/log"

	proxy "github.com/yeqown/fasthttp-reverse-proxy"
	"infini.sh/framework/lib/fasthttp"
)

var (
	proxyServer  = proxy.NewReverseProxy("localhost:8080")
	proxyServer2 = proxy.NewReverseProxy("api-js.mixpanel.com")
	proxyServer3 = proxy.NewReverseProxy("baidu.com")
)

// ProxyHandler ... fasthttp.RequestHandler func
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	requestURI := string(ctx.RequestURI())
	log.Info("requestURI=", requestURI)

	if strings.HasPrefix(requestURI, "/local") {
		// "/local" path proxy to localhost
		proxyServer.ServeHTTP(ctx)
	} else if strings.HasPrefix(requestURI, "/baidu") {
		proxyServer3.ServeHTTP(ctx)
	} else {
		proxyServer2.ServeHTTP(ctx)
	}
}

func main() {
	if err := fasthttp.ListenAndServe(":8081", ProxyHandler); err != nil {
		log.Fatal(err)
	}
}
