// Copyright 2018 The yeqown Author. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package elastic

import (
	"crypto/tls"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/config"
	"infini.sh/gateway/proxy/balancer"
	"math/rand"
	"net/http"
)

type ReverseProxy struct {
	oldAddr     string
	bla         balancer.IBalancer
	clients     []*fasthttp.HostClient
	proxyConfig *config.ProxyConfig
}

func NewReverseProxy(cfg *config.ProxyConfig) *ReverseProxy {

	//ups := config.GetUpstreamConfigs()
	//ups := config.GetActiveUpstreamConfigs()

	//log.Trace("active upstream: ", ups)

	p := ReverseProxy{
		oldAddr:     "",
		clients:     []*fasthttp.HostClient{},
		proxyConfig: cfg,
	}

	ws := []int{}

	esConfig := elastic.GetConfig(cfg.Elasticsearch)
	endpoints:=[]string{}

	if cfg.Discover.Enabled {
		nodes, err := elastic.GetClient(esConfig.Name).GetNodes()
		if err!=nil{
			panic(err)
		}

		for _,y:=range nodes.Nodes{
			endpoint:=y.(map[string]interface{})["http"].(map[string]interface{})["publish_address"].(string)
			endpoints=append(endpoints,endpoint)
		}
		log.Infof("discovery %v nodes: [%v]",len(nodes.Nodes),util.JoinArray(endpoints,", "))
	}else{
		endpoints=append(endpoints,esConfig.GetHost())
	}

	for _,endpoint:=range endpoints{

		client := &fasthttp.HostClient{
			Addr:                          endpoint,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			MaxConns:                      cfg.MaxConnection,
			MaxResponseBodySize:           cfg.MaxResponseBodySize,
			//TODO
			//MaxConnWaitTimeout: cfg.MaxConnWaitTimeout,
			//MaxConnDuration: cfg.MaxConnWaitTimeout,
			//MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			//ReadTimeout: cfg.ReadTimeout,
			//WriteTimeout: cfg.ReadTimeout,
			//ReadBufferSize: cfg.ReadBufferSize,
			//WriteBufferSize: cfg.WriteBufferSize,
			//RetryIf: func(request *fasthttp.Request) bool {
			//
			//},
			IsTLS:                         esConfig.IsTLS(),
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, //TODO
			},
		}

		p.clients=append(p.clients,client)

		//get predefined weights
		w,o:=cfg.Weights[endpoint]
		if !o||w<=0{
			w=1
		}
		ws=append(ws,w)
	}

	if len(p.clients)==0{
		panic(errors.New("proxy upstream is empty"))
	}

	p.bla = balancer.NewBalancer(ws)
	return &p
}

func (p *ReverseProxy) getClient() *fasthttp.HostClient {
	if p.clients == nil {
		panic("ReverseProxy has been closed")
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		c:= p.clients[idx]
		return c
	}

	//or go random way
	max:=len(p.clients)
	seed:=rand.Intn(max)
	c:= p.clients[seed]
	return c
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
}

//// Delegate ReverseProxy to serve
//// ref to: https://golang.org/src/net/http/httputil/reverseproxy.go#L169
//func (p *ReverseProxy) DelegateToUpstream(ctx *fasthttp.RequestCtx) {
//
//	req := &ctx.Request
//	res := &ctx.Response
//	res.Reset()
//
//	//if ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
//	//	if global.Env().IsDebug {
//	//		log.Trace("requesting from:", ctx.RemoteAddr(), ",id:", ctx.ID(), " , method:", string(ctx.Method()), ", TLS:", ctx.IsTLS())
//	//	}
//	//	req.Header.Add("X-Forwarded-For", ip)
//	//}
//
//	////routing by domain
//	//{
//	//	host := req.Header.Host() //访问的请求所对应的主机或域名,非访客地址,如: localhost:8080
//	//	if global.Env().IsDebug{
//	//		log.Trace("host, ",string(host))
//	//	}
//	//}
//
//
//
//	//cleanHopHeaders(req)
//
//	//method := string(req.Header.Method())
//	//url := string(req.RequestURI())
//	//args := req.URI().QueryArgs()
//	//
//	//if global.Env().IsDebug {
//	//	fmt.Println(method, ",", url, ",", args)
//	//}
//
//	//stats.Increment("request", strings.ToLower(strings.TrimSpace(method)))
//
//	//cacheable := false
//	//
//	//if string(req.Header.Method()) == fasthttp.MethodGet {
//	//	cacheable = true
//	//}
//
//	////check special path
//	//switch {
//	////case url == "/favicon.ico":
//	////	ctx.Response.SetStatusCode(http.StatusNotFound)
//	////	return
//	//case util.ContainStr(url, "/_search"):
//	//	//if util.ContainStr(url, "*") {
//	//	//	//fmt.Println("hit index pattern")
//	//	//	//GET _cat/indices/filebeat-*?s=index:desc
//	//	//}
//	//	cacheable = true
//	//	break
//	//case util.ContainsAnyInArray(url, []string{"_mget", "/_security/user/_has_privileges", ".kibana_task_manager/_update_by_query", "/.kibana/_update/search-telemetry", "/.kibana/_update/ui-metric"}):
//	//	//TODO get TTL config, various by request, throttle request from various clients, but doing same work
//	//	cacheable = true
//	//	break
//	//case util.ContainStr(url, "_async_search"):
//	//
//	//	if method == fasthttp.MethodPost {
//	//		//request normalization
//	//		//timestamp precision processing, scale time from million seconds to seconds, for cache reuse, for search optimization purpose
//	//		//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
//	//		//==>
//	//		//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:00.000Z","format":"strict_date_optional_time"}
//	//		body := req.Body()
//	//		log.Debug("timestamp precision updaing,", string(body))
//	//
//	//		//TODO get time field from index pattern settings
//	//		ok := util.ProcessJsonData(&body, []byte("@timestamp"), []byte("strict_date_optional_time"), []byte("range"), true, func(start, end int) {
//	//			startProcess := false
//	//			precisionLimit := 4 //0-9: 时分秒微妙 00:00:00:000
//	//			precisionOffset := 0
//	//			for i, v := range body[start:end] {
//	//				if v == 84 {
//	//					startProcess = true
//	//					precisionOffset = 0
//	//					continue
//	//				}
//	//				if startProcess && v > 47 && v < 58 {
//	//					precisionOffset++
//	//					if precisionOffset <= precisionLimit {
//	//						continue
//	//					} else if precisionOffset > 9 {
//	//						startProcess = false
//	//						continue
//	//					}
//	//					body[start+i] = 48
//	//				}
//	//
//	//			}
//	//		})
//	//		if ok {
//	//			req.SetBody(body)
//	//			log.Trace("timestamp precision updated,", string(body))
//	//		}
//	//
//	//		//{"size":0,"query":{"bool":{"must":[{"range":{"@timestamp":{"gte":"2019-09-26T15:16:59.127Z","lte":"2020-09-26T15:16:59.127Z","format":"strict_date_optional_time"}}}],"filter":[{"match_all":{}}],"should":[],"must_not":[]}},"aggs":{"61ca57f1-469d-11e7-af02-69e470af7417":{"terms":{"field":"log.file.path","order":{"_count":"desc"}},"aggs":{"timeseries":{"date_histogram":{"field":"@timestamp","min_doc_count":0,"time_zone":"Asia/Shanghai","extended_bounds":{"min":1569511019127,"max":1601133419127},"fixed_interval":"86400s"},"aggs":{"61ca57f2-469d-11e7-af02-69e470af7417":{"bucket_script":{"buckets_path":{"count":"_count"},"script":{"source":"count * 1","lang":"expression"},"gap_policy":"skip"}}}}},"meta":{"timeField":"@timestamp","intervalString":"86400s","bucketSize":86400,"seriesId":"61ca57f1-469d-11e7-af02-69e470af7417"}}},"timeout":"30000ms"}
//	//
//	//	}
//	//	cacheable = true
//	//	break
//	//}
//	//
//	////check bypass patterns
//	//if util.ContainsAnyInArray(url, p.proxyConfig.PassPatterns) {
//	//	if global.Env().IsDebug {
//	//		log.Trace("url hit bypass pattern, will not be cached, ", url)
//	//	}
//	//	cacheable = false
//	//}
//	//
//	//if args.Has("no_cache"){
//	//	cacheable=false
//	//	req.URI().QueryArgs().Del("no_cache")
//	//}
//
//
//
//
//	p.DelegateRequest(req, res)
//
//	//switch method {
//	//case fasthttp.MethodGet:
//	//
//	//	p.DelegateRequest(req, res)
//	//
//	//	break
//	//case fasthttp.MethodPost:
//	//	if p.HandleIndex(ctx) {
//	//		break
//	//	}
//	//
//	//	p.DelegateRequest(req, res)
//	//	break
//	//case fasthttp.MethodPut:
//	//	//处理索引请求
//	//	if p.HandleIndex(ctx) {
//	//		break
//	//	}
//	//
//	//	p.DelegateRequest(req, res)
//	//	break
//	//case fasthttp.MethodDelete:
//	//	p.DelegateRequest(req, res)
//	//	break
//	//default:
//	//	if global.Env().IsDebug {
//	//		log.Trace("hit default method")
//	//	}
//	//	p.DelegateRequest(req, res)
//	//}
//}

func (p *ReverseProxy) DelegateRequest(req *fasthttp.Request, res *fasthttp.Response) {

	stats.Increment("cache", "strike")

	//使用算法来获取合适的 client
	pc := p.getClient()

	cleanHopHeaders(req)

	if global.Env().IsDebug{
		log.Tracef("send request [%v] to upstream [%v]",req.URI().String(),pc.Addr)
	}


	if err := pc.Do(req, res); err != nil {
		log.Errorf("failed to proxy request: %v, %v", err, string(req.RequestURI()))
		res.SetStatusCode(http.StatusInternalServerError)
		res.SetBodyRaw([]byte(err.Error()))
	}

	res.Header.Set("UPSTREAM",pc.Addr)

}

// SetClient ...
func (p *ReverseProxy) SetClient(addr string) *ReverseProxy {
	for idx := range p.clients {
		p.clients[idx].Addr = addr
	}
	return p
}

// Reset ...
func (p *ReverseProxy) Reset() {
	for idx := range p.clients {
		p.clients[idx].Addr = ""
	}
}

// Close ... clear and release
func (p *ReverseProxy) Close() {
	p.clients = nil
	p.bla = nil
	p = nil
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",          // Connection
	"Proxy-Connection",    // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",          // Keep-Alive
	"Proxy-Authenticate",  // Proxy-Authenticate
	"Proxy-Authorization", // Proxy-Authorization
	"Te",                  // canonicalized version of "TE"
	"Trailer",             // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",   // Transfer-Encoding
	"Upgrade",             // Upgrade

	//"Accept-Encoding",             // Disable Gzip
	//"Content-Encoding",             // Disable Gzip
}
