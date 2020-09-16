// Copyright 2018 The yeqown Author. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/dgraph-io/ristretto"
	"github.com/valyala/fasthttp"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/config"
	"infini.sh/gateway/translog"
	"net"
	"net/http"
	"src/github.com/go-redis/redis"
	"strings"
	"sync"
)

// ReverseProxy reverse handler using fasthttp.HostClient
// TODO: support https config
type ReverseProxy struct {
	oldAddr string                 // old addr to keep old API working as usual
	bla     IBalancer              // balancer
	clients []*fasthttp.HostClient // clients
}

var proxyConfig *config.ProxyConfig

//var proxyCache *ccache.Cache

var client *redis.Client
var cache *ristretto.Cache

func getRedisClient() *redis.Client {
	l.Lock()
	defer l.Unlock()
	if client != nil {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "localhost", "6379"),
		//Password: handler.config.RedisConfig.Password,
		//DB:       handler.config.RedisConfig.DB,
		Password: "",
		DB:       0,
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)

	return client
}

// NewReverseProxy create an ReverseProxy with options
func NewReverseProxy(cfg *config.ProxyConfig) *ReverseProxy {

	var err error
	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Num keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}


	proxyConfig = cfg
	//proxyCache = ccache.New(ccache.Configure().MaxSize(proxyConfig.CacheConfig.MaxCachedItem).ItemsToPrune(100))

	ups := config.GetActiveUpstreamConfigs()

	log.Trace("active upstream: ", ups)

	//// apply an new object of `ReverseProxy`
	p := ReverseProxy{
		oldAddr: "",
		clients: make([]*fasthttp.HostClient, len(ups)),
	}

	ws := make([]int, len(ups))

	//TODO handle disable or inactive case
	i := 0
	for k, v := range ups {
		log.Tracef("parse upstream: %s , config: %v", k, v)

		if v.Weight <= 0 {
			v.Weight = 1
		}

		ws[i] = v.Weight
		esConfig := elastic.GetConfig(v.Elasticsearch)
		log.Trace("es config, ", esConfig)
		p.clients[i] = &fasthttp.HostClient{
			Addr:                          esConfig.GetHost(),
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			MaxConns:                      v.MaxConnection,
			MaxResponseBodySize:           20 * 1024 * 1024,
			IsTLS:                         esConfig.IsTLS(),
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		i++
	}

	p.bla = NewBalancer(ws)
	return &p

}

func (p *ReverseProxy) getClient() *fasthttp.HostClient {
	if p.clients == nil {
		// closed
		panic("ReverseProxy has been closed")
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		return p.clients[idx]
	}

	return p.clients[0]
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		// if h == "Te" && hv == "trailers" {
		// 	continue
		// }
		req.Header.Del(h)
	}
}

var jsonOK = "{ \"took\" : 1, \"errors\" : false }"
var bulkRequestOKBody = []byte(jsonOK)

func (p *ReverseProxy) HandleIndex(ctx *fasthttp.RequestCtx) bool {
	if global.Env().IsDebug {
		log.Trace("try to handle index operations")
	}
	//bulk
	//index

	if strings.Contains(ctx.URI().String(), "_bulk") {

		stats.Increment("requests", "in.bulk")

		if global.Env().IsDebug {
			log.Trace("saving bulk request")
		}

		translog.SaveRequest(ctx)

		ctx.Response.SetStatusCode(http.StatusOK)
		ctx.Response.SwapBody(bulkRequestOKBody)

		return true
	}

	return false
}

func (p *ReverseProxy) getHash(req *fasthttp.Request) string {
	buffer := bytes.NewBuffer(req.Body())
	buffer.Write(req.Header.Method())
	buffer.Write(req.RequestURI())
	buffer.Write(req.URI().QueryArgs().QueryString())
	buffer.Write(req.Body())
	return util.MD5digestString(buffer.Bytes())
}

var byPassGet = []string{"scroll", "scroll_id"}

//var v = map[string][]byte{}
var l sync.RWMutex

func Get(key string) []byte {

	o,found:=cache.Get(key)
	if found {
		return o.([]byte)
	}
	return nil

	//b, err := getRedisClient().Get(key).Result()
	//if err == redis.Nil {
	//	fmt.Println(key, " 不存在")
	//	return nil
	//} else if err != nil {
	//	panic(err)
	//} else {
	//	//fmt.Println(key, ":", string(b))
	//}
	//return []byte(b)

	//l.Lock()
	//defer l.Unlock()
	//return v[key]

}
func Set(key string, data []byte) {

	cache.Set(key, data, 1) // set a value


	//err := getRedisClient().Set(key, data, 100*time.Second).Err()
	//if err != nil {
	//	panic(err)
	//}

	//l.Lock()
	//defer l.Unlock()
	// v[key]=data
}

// ServeHTTP ReverseProxy to serve
// ref to: https://golang.org/src/net/http/httputil/reverseproxy.go#L169
func (p *ReverseProxy) ServeHTTP(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	res := &ctx.Response

	res.Reset()

	//TODO 根据请求IP和头信息,执行请求拒绝, 基于后台设置的黑白名单,执行准入, 只允许特定 IP Agent 访问 Gateway 访问

	//TODO 慢查询,非法查询 主动检测和拒绝, 走 cache

	//TODO 记录所有请求,采样记录,按条件记录

	//自动学习请求网站来生成 FST 路由信息, 基于 FST 数来快速路由

	// prepare request(replace headers and some URL host)
	if ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		if global.Env().IsDebug {
			log.Trace("requesting from:", ctx.RemoteAddr(), ",id:", ctx.ID(), " , method:", string(ctx.Method()), ", TLS:", ctx.IsTLS())
		}
		req.Header.Add("X-Forwarded-For", ip)
	}

	// to save all response header
	// resHeaders := make(map[string]string)
	// res.Header.VisitAll(func(k, v []byte) {
	// 	key := string(k)
	// 	value := string(v)
	// 	if val, ok := resHeaders[key]; ok {
	// 		resHeaders[key] = val + "," + value
	// 	}
	// 	resHeaders[key] = value
	// })

	////routing by domain
	//{
	//	host := req.Header.Host() //访问的请求所对应的主机或域名,非访客地址,如: localhost:8080
	//	if global.Env().IsDebug{
	//		log.Trace("host, ",string(host))
	//	}
	//}

	//使用算法来获取合适的 client
	pc := p.getClient()

	// assign the host to support virtual hosting, aka shared web hosting (one IP, multiple domains)
	req.SetHost(pc.Addr)

	cleanHopHeaders(req)

	// Check that the server actually sent compressed data
	//var reader io.ReadCloser
	//var err error
	var useGzip bool
	if req.Header.HasAcceptEncoding("gzip") {
		useGzip = true
		//fmt.Println("use gzip")
	}

	method := string(req.Header.Method())
	url := string(req.RequestURI())
	args := req.URI().QueryArgs()

	//if global.Env().IsDebug {
		fmt.Println(method, ",", url, ",", args)
	//}

	//fmt.Println(req.PostArgs()) //curl -XPOST http://localhost:8000/_search -d'a=b'

	stats.Increment("requests", strings.ToLower(strings.TrimSpace(method)))

	cacheable := false

	if string(req.Header.Method()) == fasthttp.MethodGet {
		cacheable = true
	}

	switch {
	case url == "/favicon.ico":
		ctx.Response.SetStatusCode(http.StatusNotFound)
		return
	case util.SuffixStr(url, "/_search"):

		//fmt.Println("hit search,", url, ",", string(req.Body()))

		if util.ContainStr(url, "*") {
			fmt.Println("hit index pattern")
			//GET _cat/indices/filebeat-*?s=index:desc

		}
		cacheable = true
		//fmt.Println("hit search")
		break
	}

	if util.ContainsAnyInArray(url, byPassGet) {
		if global.Env().IsDebug {
			log.Trace("url hit bypass pattern, will not be cached, ", url)
		}
		cacheable = false
	}

	//TODO optimize scroll API, should always point to same IP, prefer to route to where index/shard located

	if cacheable && proxyConfig.CacheConfig.Enabled {

		//LRU 缓存可以选择开启
		//5s 内,如果相同的 hash 出现过 2 次,则缓存起来第 3 次, 有效期 10s
		//hash->count, hash->content

		hash := p.getHash(req)
		//hash_type := hash+".type"

		//item := proxyCache.Get(hash)
		item := Get(hash)
		//if global.Env().IsDebug {
			fmt.Println("hash,", hash,len(item))
		//}

		if item == nil {
			//handle
			if global.Env().IsDebug {
				log.Trace("cache miss, ", hash)
			}

			stats.Increment("cache", "miss")

			//always gzip between es and gateway
			if useGzip {
				req.Header.Set("Accept-Encoding", "gzip")
			}

			if err := pc.Do(req, res); err != nil {
				log.Errorf("failed to proxy request: %v\n", err)
				res.SetStatusCode(http.StatusInternalServerError)
				res.SwapBody([]byte(err.Error()))
				return
			}

			body := res.Body()

			//cache 200 only
			if res.StatusCode() == http.StatusOK {

				//var o = map[string]interface{}{}

				//err := util.FromJson(string(res.BodyInflate()), &o)
				//if err != nil {
				//	fmt.Println("set cache json error,",string(body))
				//	panic(err)
				//}
				//proxyCache.Set(hash, body, proxyConfig.CacheConfig.GetTTLDuration())
				contentType := res.Header.ContentType()
				if global.Env().IsDebug {
					fmt.Println("resp content type:", string(contentType))
				}
				Set(hash, body)

				//proxyCache.Set(hash_type,contentType , proxyConfig.CacheConfig.GetTTLDuration())
			} else {
				if global.Env().IsDebug {
					fmt.Println(res.StatusCode())
					fmt.Println(string(req.RequestURI()))
					//fmt.Println(string(res.Body()))
				}
			}
			if global.Env().IsDebug {
				fmt.Println("cache missing")
			}
			return

		} else if len(item) > 0 {
			//content := item
			//fmt.Println("content:",item,",",len(item))
			////content := item.Value().([]byte)
			//if global.Env().IsDebug {
			//	log.Trace("hit cache, ", hash, ", expired: ", item.Expired(), ", ttl:", item.TTL())
			//}

			stats.Increment("cache", "hit")

			//if TTL <1s, fetch in background
			//go func() {
			//trigger cached refresh in tasks
			//}()

			//res.Header.Set("TTL", item.TTL().String())

			res.Header.DisableNormalizing()

			//contentType := proxyCache.Get(hash_type)
			//if contentType!=nil{
			//	fmt.Println("hit:",string(contentType.Value().([]byte)))
			//	res.Header.SetBytesV("Content-Type",contentType.Value().([]byte))
			//}else{
			if util.PrefixStr(url, "/_cat") {
				res.Header.SetBytesV("content-type", []byte("text/plain"))
			} else {
				res.Header.SetBytesV("content-type", []byte("application/json"))
			}

			if useGzip {
				fmt.Println("use gzip")
				//TODO, if request not asking gzip, ungzip first befrore return
				res.Header.Set("content-encoding", "gzip")
			}

			var o = map[string]interface{}{}
			if global.Env().IsDebug {
				err := util.FromJson(string(item), &o)
				if err != nil {
					fmt.Println("json error", string(req.Header.Method()), string(res.Header.ContentType()), res.StatusCode(), string(req.RequestURI()), string(item))

					panic(err)
				}
			}
			//fmt.Println(string(item))
			//output cached response
			res.SetStatusCode(http.StatusOK)
			res.Header.SetContentLength(len(item))

			var body []byte
			body = append(body, item...) //need to copy it into a memory before exit the Handle function
			//ctx.SetBody(body)

			res.ResetBody()
			res.SetBodyRaw(body)


			fmt.Println("cache req:",req.Header.String())
			fmt.Println("cache res:",res.Header.String())
			//fmt.Println(string(item))


			if global.Env().IsDebug {
				fmt.Println("hit cache,", hash, ",", string(req.RequestURI()), ",", len(item))
			}
			//async delete
			//if item.Expired() {
			//	if global.Env().IsDebug {
			//		log.Trace("cache expired, release now, ", hash)
			//	}
			//	//item.Release()
			//	proxyCache.Delete(hash)
			//	//proxyCache.Delete(hash_type)
			//	stats.Increment("cache", "hit_expired")
			//}

			return
		}

	}

	switch method {
	case fasthttp.MethodGet:

		if global.Env().IsDebug {
			log.Trace("hit get method")
		}

		//handler.handleRead(w, req, body)

		p.delegateRequest(req, res)
		if global.Env().IsDebug {
			fmt.Println("hit last get")
		}

		break
	case fasthttp.MethodPost:
		if p.HandleIndex(ctx) {
			break
		}

		p.delegateRequest(req, res)

		break
	case fasthttp.MethodPut:
		//可能是写
		//排除部分读和查询请求(_search)
		//handler.handleWrite(w, req, body)

		//处理索引请求
		if p.HandleIndex(ctx) {
			break
		}

		p.delegateRequest(req, res)

		break
	case fasthttp.MethodDelete:
		//handler.handleWrite(w, req, body)
		p.delegateRequest(req, res)

		break
	default:
		if global.Env().IsDebug {
			log.Trace("hit default method")
		}
		p.delegateRequest(req, res)
	}

	//logger.Debugf("response headers = %s", res.Header.String())
	// write response headers
	//for _, h := range hopHeaders {
	//	res.Header.Del(h)
	//}

	// logger.Debugf("response headers = %s", resHeaders)
	// for k, v := range resHeaders {
	// 	res.Header.Set(k, v)
	// }
}

func (p *ReverseProxy) delegateRequest(req *fasthttp.Request, res *fasthttp.Response) {
	if global.Env().IsDebug {
		fmt.Println("delegrate request,", string(req.Header.Method()), string(req.RequestURI()), ",", string(req.Body()))
	}
	if global.Env().IsDebug {
		log.Trace("delegate request by default")
	}

	//使用算法来获取合适的 client
	pc := p.getClient()

	// assign the host to support virtual hosting, aka shared web hosting (one IP, multiple domains)
	req.SetHost(pc.Addr)

	cleanHopHeaders(req)

	if err := pc.Do(req, res); err != nil {
		log.Errorf("failed to proxy request: %v\n", err)
		res.SetStatusCode(http.StatusInternalServerError)
		res.SwapBody([]byte(err.Error()))
		return
	}

	//verify content type
	if global.Env().IsDebug {
		if string(req.Header.Method()) != fasthttp.MethodHead {
			var o = map[string]interface{}{}

			err := util.FromJson(string(res.Body()), &o)
			if err != nil {
				fmt.Println("json error", string(req.Header.Method()), string(res.Header.ContentType()), res.StatusCode(), string(req.RequestURI()), string(res.Body()))
				panic(err)
			}
		}
	}

	fmt.Println("cold req:",req.Header.String())
	fmt.Println("cold res:",res.Header.String())


	res.SetStatusCode(http.StatusOK)
	res.SwapBody(res.Body())
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
	//p.opt = nil
	p.bla = nil
	//p.ws = nil
	p = nil
}

//
//func copyResponse(src *fasthttp.Response, dst *fasthttp.Response) {
//	src.CopyTo(dst)
//	logger.Debugf("response header=%v", src.Header)
//}
//
//func copyRequest(src *fasthttp.Request, dst *fasthttp.Request) {
//	src.CopyTo(dst)
//}
//
//func cloneResponse(src *fasthttp.Response) *fasthttp.Response {
//	dst := new(fasthttp.Response)
//	copyResponse(src, dst)
//	return dst
//}
//
//func cloneRequest(src *fasthttp.Request) *fasthttp.Request {
//	dst := new(fasthttp.Request)
//	copyRequest(src, dst)
//	return dst
//}

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
	//"Accept-Encoding",
}
