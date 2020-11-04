// Copyright 2018 The yeqown Author. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/dgraph-io/ristretto"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/cache"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/config"
	"infini.sh/gateway/proxy/balancer"
	"infini.sh/gateway/proxy/output/translog"
	"math/rand"
	"net/http"
	"src/github.com/go-redis/redis"
	"strings"
	"sync"
	"time"
)

// TODO: support https config
type ReverseProxy struct {
	oldAddr     string
	bla         balancer.IBalancer
	clients     []*fasthttp.HostClient
	proxyConfig *config.ProxyConfig
}

var ccacheCache *ccache.LayeredCache

var client *redis.Client
var cache *ristretto.Cache

func getRedisClient() *redis.Client {

	if client != nil {
		return client
	}

	l.Lock()
	defer l.Unlock()

	if client != nil {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "192.168.3.98", "6379"),
		//Password: handler.config.RedisConfig.Password,
		//DB:       handler.config.RedisConfig.DB,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}

	return client
}

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

	ccacheCache = ccache.Layered(ccache.Configure().MaxSize(cfg.CacheConfig.MaxCachedItem).ItemsToPrune(100))

	ups := config.GetActiveUpstreamConfigs()

	log.Trace("active upstream: ", ups)

	p := ReverseProxy{
		oldAddr:     "",
		clients:     []*fasthttp.HostClient{},
		proxyConfig: cfg,
	}

	ws := []int{}

	//TODO handle disable or inactive case
	i := 0
	for k, v := range ups {
		log.Tracef("parse upstream: %s , config: %v", k, v)

		if v.Weight <= 0 {
			v.Weight = 1
		}

		esConfig := elastic.GetConfig(v.Elasticsearch)

		if v.DiscoveryNodes{
			nodes,err:=elastic.GetClient(v.Elasticsearch).GetNodes()
			if err!=nil{
				panic(err)
			}
			//fmt.Println(nodes.ClusterName)
			for _,y:=range nodes.Nodes{
				//fmt.Println(x)
				//fmt.Println(y.(map[string]interface{})["name"])
				////fmt.Println(y.(map[string]interface{})["host"])
				//fmt.Println(y.(map[string]interface{})["ip"])
				//fmt.Println(y.(map[string]interface{})["version"])
				//fmt.Println(y.(map[string]interface{})["roles"])
				endpoint:=y.(map[string]interface{})["http"].(map[string]interface{})["publish_address"]
				//fmt.Println(endpoint)
				////fmt.Println(y.(map[string]interface{})["http"].(map[string]interface{})["max_content_length_in_bytes"])
				//fmt.Println()
				log.Trace("es config, ", esConfig)
				client := &fasthttp.HostClient{
					Addr:                          endpoint.(string),
					DisableHeaderNamesNormalizing: true,
					DisablePathNormalizing:        true,
					MaxConns:                      v.MaxConnection,
					MaxResponseBodySize:           20 * 1024 * 1024,
					IsTLS:                         esConfig.IsTLS(),
					TLSConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}
				p.clients=append(p.clients,client)
				ws=append(ws,v.Weight)
				i++
			}
		}else{
			log.Trace("es config, ", esConfig)
			client := &fasthttp.HostClient{
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
			p.clients=append(p.clients,client)
			ws=append(ws,v.Weight)
			i++
		}

	}

	if len(p.clients)==0{
		panic(errors.New("upstream is not set"))
	}

	fmt.Println(len(p.clients))
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

var jsonOK = "{ \"took\" : 1, \"errors\" : false }"
var bulkRequestOKBody = []byte(jsonOK)

func (p *ReverseProxy) HandleIndex(ctx *fasthttp.RequestCtx) bool {
	if global.Env().IsDebug {
		log.Trace("try to handle index operations")
	}
	//bulk
	//index

	if p.proxyConfig.AsyncWrite && strings.Contains(ctx.URI().String(), "_bulk") {

		stats.Increment("request", "action.bulk")

		if global.Env().IsDebug {
			log.Trace("saving bulk request")
		}

		translog.SaveRequest(ctx)

		ctx.Response.SetStatusCode(http.StatusOK)
		ctx.Response.SetBodyRaw(bulkRequestOKBody)

		return true
	}

	return false
}

func (p *ReverseProxy) getHash(req *fasthttp.Request) string {

	//TODO configure, remove keys from hash factor
	req.URI().QueryArgs().Del("preference")

	data := make([]byte,
		len(req.Body())+
			len(req.URI().QueryArgs().QueryString())+
			len(req.PostArgs().QueryString())+
			len(req.Header.Method())+
			len(req.RequestURI()),
	)

	if global.Env().IsDebug {
		log.Trace("generate hash:", string(req.Header.Method()), string(req.RequestURI()), string(req.URI().QueryArgs().QueryString()), string(req.Body()), string(req.PostArgs().QueryString()))
	}

	//TODO 后台可以按照请求路径来勾选 Hash 因子

	buffer := bytes.NewBuffer(data)
	buffer.Write(req.Header.Method())
	buffer.Write(req.Header.Peek("Authorization")) //TODO enable configure for this feature, may filter by user or share, add/remove Authorization header to hash factor
	buffer.Write(req.RequestURI())
	buffer.Write(req.URI().QueryArgs().QueryString())
	buffer.Write(req.Body())
	buffer.Write(req.PostArgs().QueryString())

	return util.MD5digestString(buffer.Bytes())
}

var l sync.RWMutex

const cacheRedis = "redis"
const cacheCCache = "ccache"

func (p *ReverseProxy) Get(key string) ([]byte, bool) {

	switch p.proxyConfig.CacheConfig.Type {
	case cacheRedis:
		b, err := getRedisClient().Get(key).Result()
		if err == redis.Nil {
			return nil, false
		} else if err != nil {
			panic(err)
		}
		return []byte(b), true
	case cacheCCache:

		item := ccacheCache.GetOrCreateSecondaryCache("default").Get(key)
		if item != nil {
			data := item.Value().([]byte)
			if item.Expired() {
				stats.Increment("cache", "expired")
				ccacheCache.GetOrCreateSecondaryCache("default").Delete(key)
			}
			return data, true
		}

	default:
		o, found := cache.Get(key)
		if found {
			return o.([]byte), found
		}
	}

	return nil, false

}

func (p *ReverseProxy) Set(key string, data []byte, ttl time.Duration) {

	switch p.proxyConfig.CacheConfig.Type {
	case cacheRedis:
		err := getRedisClient().Set(key, data, ttl).Err()
		if err != nil {
			panic(err)
		}
		return
	case cacheCCache:
		ccacheCache.GetOrCreateSecondaryCache("default").Set(key, data, ttl)
		return
	default:
		cache.SetWithTTL(key, data, 1, ttl)
	}

}

var colon = []byte(": ")
var newLine = []byte("\n")

//TODO optimize memmove issue, buffer read
func (p *ReverseProxy) Decode(data []byte, req *fasthttp.Request, res *fasthttp.Response) {
	readerHeaderLengthBytes := make([]byte, 4)
	reader := bytes.NewBuffer(data)
	_, err := reader.Read(readerHeaderLengthBytes)
	if err != nil {
		panic(err)
	}

	readerHeaderLength := binary.LittleEndian.Uint32(readerHeaderLengthBytes)
	readerHeader := make([]byte, readerHeaderLength)
	_, err = reader.Read(readerHeader)
	if err != nil {
		panic(err)
	}

	line := bytes.Split(readerHeader, newLine)
	for _, l := range line {
		kv := bytes.Split(l, colon)
		if len(kv) == 2 {
			res.Header.SetBytesKV(kv[0], kv[1])
		}
	}

	readerBodyLengthBytes := make([]byte, 4)
	_, err = reader.Read(readerBodyLengthBytes)
	if err != nil {
		panic(err)
	}

	readerBodyLength := binary.LittleEndian.Uint32(readerBodyLengthBytes)
	readerBody := make([]byte, readerBodyLength)
	_, err = reader.Read(readerBody)
	if err != nil {
		panic(err)
	}

	res.SetBodyRaw(readerBody)
	res.SetStatusCode(fasthttp.StatusOK)
}

func (p *ReverseProxy) Encode(req *fasthttp.Request, res *fasthttp.Response) []byte {

	buffer := bytes.Buffer{}
	res.Header.VisitAll(func(key, value []byte) {
		buffer.Write(key)
		buffer.Write(colon)
		buffer.Write(value)
		buffer.Write(newLine)
	})

	header := buffer.Bytes()
	body := res.Body()

	data := bytes.Buffer{}

	headerLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerLength, uint32(len(header)))

	bodyLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(bodyLength, uint32(len(body)))

	//header length
	data.Write(headerLength)
	data.Write(header)

	//body
	data.Write(bodyLength)
	data.Write(body)

	return data.Bytes()
}

// Delegate ReverseProxy to serve
// ref to: https://golang.org/src/net/http/httputil/reverseproxy.go#L169
func (p *ReverseProxy) DelegateToUpstream(ctx *fasthttp.RequestCtx) {

	req := &ctx.Request
	res := &ctx.Response
	res.Reset()

	//if ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
	//	if global.Env().IsDebug {
	//		log.Trace("requesting from:", ctx.RemoteAddr(), ",id:", ctx.ID(), " , method:", string(ctx.Method()), ", TLS:", ctx.IsTLS())
	//	}
	//	req.Header.Add("X-Forwarded-For", ip)
	//}

	////routing by domain
	//{
	//	host := req.Header.Host() //访问的请求所对应的主机或域名,非访客地址,如: localhost:8080
	//	if global.Env().IsDebug{
	//		log.Trace("host, ",string(host))
	//	}
	//}



	cleanHopHeaders(req)

	method := string(req.Header.Method())
	url := string(req.RequestURI())
	args := req.URI().QueryArgs()

	if global.Env().IsDebug {
		fmt.Println(method, ",", url, ",", args)
	}

	stats.Increment("request", strings.ToLower(strings.TrimSpace(method)))

	cacheable := false

	if string(req.Header.Method()) == fasthttp.MethodGet {
		cacheable = true
	}

	//check special path
	switch {
	case url == "/favicon.ico":
		ctx.Response.SetStatusCode(http.StatusNotFound)
		return
	case util.ContainStr(url, "/_search"):
		//if util.ContainStr(url, "*") {
		//	//fmt.Println("hit index pattern")
		//	//GET _cat/indices/filebeat-*?s=index:desc
		//}
		cacheable = true
		break
	case util.ContainsAnyInArray(url, []string{"_mget", "/_security/user/_has_privileges", ".kibana_task_manager/_update_by_query", "/.kibana/_update/search-telemetry", "/.kibana/_update/ui-metric"}):
		//TODO get TTL config, various by request, throttle request from various clients, but doing same work
		cacheable = true
		break
	case util.ContainStr(url, "_async_search"):

		if method == fasthttp.MethodPost {
			//request normalization
			//timestamp precision processing, scale time from million seconds to seconds, for cache reuse, for search optimization purpose
			//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
			//==>
			//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:00.000Z","format":"strict_date_optional_time"}
			body := req.Body()
			log.Debug("timestamp precision updaing,", string(body))

			//TODO get time field from index pattern settings
			ok := util.ProcessJsonData(&body, []byte("@timestamp"), []byte("strict_date_optional_time"), []byte("range"), true, func(start, end int) {
				startProcess := false
				precisionLimit := 4 //0-9: 时分秒微妙 00:00:00:000
				precisionOffset := 0
				for i, v := range body[start:end] {
					if v == 84 {
						startProcess = true
						precisionOffset = 0
						continue
					}
					if startProcess && v > 47 && v < 58 {
						precisionOffset++
						if precisionOffset <= precisionLimit {
							continue
						} else if precisionOffset > 9 {
							startProcess = false
							continue
						}
						body[start+i] = 48
					}

				}
			})
			if ok {
				req.SetBody(body)
				log.Trace("timestamp precision updated,", string(body))
			}

			//{"size":0,"query":{"bool":{"must":[{"range":{"@timestamp":{"gte":"2019-09-26T15:16:59.127Z","lte":"2020-09-26T15:16:59.127Z","format":"strict_date_optional_time"}}}],"filter":[{"match_all":{}}],"should":[],"must_not":[]}},"aggs":{"61ca57f1-469d-11e7-af02-69e470af7417":{"terms":{"field":"log.file.path","order":{"_count":"desc"}},"aggs":{"timeseries":{"date_histogram":{"field":"@timestamp","min_doc_count":0,"time_zone":"Asia/Shanghai","extended_bounds":{"min":1569511019127,"max":1601133419127},"fixed_interval":"86400s"},"aggs":{"61ca57f2-469d-11e7-af02-69e470af7417":{"bucket_script":{"buckets_path":{"count":"_count"},"script":{"source":"count * 1","lang":"expression"},"gap_policy":"skip"}}}}},"meta":{"timeField":"@timestamp","intervalString":"86400s","bucketSize":86400,"seriesId":"61ca57f1-469d-11e7-af02-69e470af7417"}}},"timeout":"30000ms"}

		}
		cacheable = true
		break
	}

	//check bypass patterns
	if util.ContainsAnyInArray(url, p.proxyConfig.PassthroughPatterns) {
		if global.Env().IsDebug {
			log.Trace("url hit bypass pattern, will not be cached, ", url)
		}
		cacheable = false
	}

	if args.Has("no_cache"){
		cacheable=false
		req.URI().QueryArgs().Del("no_cache")
	}

	//TODO optimize scroll API, should always point to same IP, prefer to route to where index/shard located

	if cacheable && p.proxyConfig.CacheConfig.Enabled {

		//LRU 缓存可以选择开启
		//5s 内,如果相同的 hash 出现过 2 次,则缓存起来第 3 次, 有效期 10s
		//hash->count, hash->content

		hash := p.getHash(req)
		item, found := p.Get(hash)

		if found {
			stats.Increment("cache", "hit")

			ctx.Response.Cached=true

			p.Decode(item, req, res)

			res.Header.DisableNormalizing()
			if global.Env().IsDebug {
				log.Trace("cache hit:", hash, ",", string(req.Header.Method()), ",", string(req.RequestURI()))
			}

			return
		} else {
			stats.Increment("cache", "miss")

			if global.Env().IsDebug {
				log.Trace("cache miss:", hash, ",", string(req.Header.Method()), ",", string(req.RequestURI()), ",", string(req.Body()))
			}

			p.DelegateRequest(req,res)

			////使用算法来获取合适的 client
			//pc := p.getClient()
			//// assign the host to support virtual hosting, aka shared web hosting (one IP, multiple domains)
			//req.SetHost(pc.Addr)
			//req.Header.Set("Host", pc.Addr)
			//if err := pc.Do(req, res); err != nil {
			//	log.Errorf("failed to proxy request: %v\n", err)
			//	res.SetStatusCode(http.StatusInternalServerError)
			//	res.SetBodyRaw([]byte(err.Error()))
			//	return
			//}

			//cache 200 only TODO allow configure to support: 404/200/201/500, also set TTL
			if res.StatusCode() == http.StatusOK {
				body := res.Body()
				var id string
				if strings.Contains(url, "/_async_search") {

					ok, b := util.ExtractFieldFromJson(&body, []byte("\"id\""), []byte("\"is_partial\""), []byte("id\""))
					if ok {

						b = bytes.Replace(b, []byte(":"), nil, -1)
						b = bytes.Replace(b, []byte("\""), nil, -1)
						b = bytes.Replace(b, []byte(","), nil, -1)
						b = bytes.TrimSpace(b)

						id = string(b)
					}

					//store cache_token
					if method == fasthttp.MethodPost {
						//TODO set the cache TTL, 30minutes
						//if response contains:
						//"id" : "FktyZXA2bklVU2VDeWIwVWdkVTlMcGcdMWpuRkM3SDZSWWVBSTdKT1hkRDNkdzoyNDY3MjY=",
						//then it is a async task, store ID to cache, and if this task finished, associate that result to this same request
						if ok {
							if global.Env().IsDebug {
								log.Trace("async hash: set async hash cache", string(id), "=>", string(hash))
							}
							p.Set(id, []byte(hash), p.proxyConfig.CacheConfig.GetAsyncSearchTTLDuration())
						}

					} else if method == fasthttp.MethodGet {

						//only cache finished async search results
						if util.BytesSearchValue(body, []byte("is_running"), []byte(","), []byte("true")) {
							if global.Env().IsDebug {
								log.Trace("async search is still running")
							}
							return
						} else {
							//async search results finished, let's cache it
							if global.Env().IsDebug {
								log.Trace("async search results finished, let's cache it")
							}

							if ok {
								item, found := p.Get(id)
								if found {
									if global.Env().IsDebug {
										log.Trace("found request hash, set cache:", id, ": ", string(item))
									}
									cacheBytes := p.Encode(req, res)
									p.Set(string(item), cacheBytes, p.proxyConfig.CacheConfig.GetChaosTTLDuration())
								} else {
									if global.Env().IsDebug {
										log.Trace("async search request hash was lost:", id)
									}
								}
							}

						}
					}

				} else {
					cacheBytes := p.Encode(req, res)
					p.Set(hash, cacheBytes, p.proxyConfig.CacheConfig.GetChaosTTLDuration())
				}

			}

			return
		}
	}

	switch method {
	case fasthttp.MethodGet:

		p.DelegateRequest(req, res)

		break
	case fasthttp.MethodPost:
		if p.HandleIndex(ctx) {
			break
		}

		p.DelegateRequest(req, res)
		break
	case fasthttp.MethodPut:
		//处理索引请求
		if p.HandleIndex(ctx) {
			break
		}

		p.DelegateRequest(req, res)
		break
	case fasthttp.MethodDelete:
		p.DelegateRequest(req, res)
		break
	default:
		if global.Env().IsDebug {
			log.Trace("hit default method")
		}
		p.DelegateRequest(req, res)
	}
}

func (p *ReverseProxy) DelegateRequest(req *fasthttp.Request, res *fasthttp.Response) {

	stats.Increment("cache", "strike")

	//使用算法来获取合适的 client
	pc := p.getClient()

	originalHost :=string(req.Host())

	cleanHopHeaders(req)

	// assign the host to support virtual hosting, aka shared web hosting (one IP, multiple domains)
	req.SetHost(pc.Addr)
	req.Header.Set("Host", pc.Addr)

	if err := pc.Do(req, res); err != nil {
		log.Errorf("failed to proxy request: %v, %v", err, string(res.Body()))
		res.SetStatusCode(http.StatusInternalServerError)
		res.SetBodyRaw([]byte(err.Error()))
	}
	req.SetHost(originalHost)

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
}
