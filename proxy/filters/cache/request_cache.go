package cache

import (
	"bytes"
	"encoding/binary"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RequestCache struct {
	param.Parameters
}

const cacheRedis = "redis"
const cacheCCache = "ccache"
const defaultCacheType = "ristretto"

var ccCache *ccache.LayeredCache
var l sync.RWMutex
var colon = []byte(": ")
var newLine = []byte("\n")
var client *redis.Client
var cache *ristretto.Cache
var inited bool

var bytesBufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (p RequestCache) getRedisClient() *redis.Client {

	if client != nil {
		return client
	}

	l.Lock()
	defer l.Unlock()

	if client != nil {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%v", p.MustGetString("redis_host"), p.MustGet("redis_port")),
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

func (p RequestCache) initCache() {
	if inited {
		return
	}

	var err error
	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,                                                // Num keys to track frequency of (10M).
		MaxCost:     p.GetInt64OrDefault("max_cached_size", 1000000000), // Maximum cost of cache (1GB).
		BufferItems: 64,                                                 // Number of keys per Get buffer.
		Metrics:     false,
	})
	if err != nil {
		panic(err)
	}

	ccCache = ccache.Layered(ccache.Configure().MaxSize(p.GetInt64OrDefault("max_cached_item", 1000000)).ItemsToPrune(100))
	inited = true
}

func (p RequestCache) GetCache(key string) ([]byte, bool) {
	p.initCache()

	switch p.GetStringOrDefault("cache_type", defaultCacheType) {
	case cacheRedis:
		b, err := p.getRedisClient().Get(key).Result()
		if err == redis.Nil {
			return nil, false
		} else if err != nil {
			panic(err)
		}
		return []byte(b), true
	case cacheCCache:

		item := ccCache.GetOrCreateSecondaryCache("default").Get(key)
		if item != nil {
			data := item.Value().([]byte)
			if item.Expired() {
				stats.Increment("cache", "expired")
				ccCache.GetOrCreateSecondaryCache("default").Delete(key)
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

func (p RequestCache) SetCache(key string, data []byte, ttl time.Duration) {

	if global.Env().IsDebug {
		log.Trace("set cache:", key, ", ttl:", ttl)
	}

	min, _ := p.GetInt("min_response_size", -1)
	max, _ := p.GetInt("max_response_size", math.MaxInt32)
	len := len(data)
	if len < min || len > max {
		if global.Env().IsDebug {
			log.Tracef("invalid response size, %v not between %v and %v", len, min, max)
		}
		return
	}

	p.initCache()

	switch p.GetStringOrDefault("cache_type", defaultCacheType) {
	case cacheRedis:
		err := p.getRedisClient().Set(key, data, ttl).Err()
		if err != nil {
			panic(err)
		}
		return
	case cacheCCache:
		ccCache.GetOrCreateSecondaryCache("default").Set(key, data, ttl)
		return
	default:
		cache.SetWithTTL(key, data, int64(len), ttl)
	}
}


//var hashBufferPool = &sync.Pool{
//	New: func() interface{} {
//		return new(bytes.Buffer)
//	},
//}

func (p RequestCache) getHash(req *fasthttp.Request) string {

	//TODO configure, remove keys from hash factor
	req.URI().QueryArgs().Del("preference")

	buffer:=bytes.Buffer{}
	//buffer:=hashBufferPool.Get().(*bytes.Buffer)

	if global.Env().IsDebug {
		log.Trace("generate hash:", string(req.Header.Method()), string(req.RequestURI()), string(req.URI().QueryArgs().QueryString()), string(req.Body()), string(req.PostArgs().QueryString()))
	}

	//TODO 后台可以按照请求路径来勾选 Hash 因子

	buffer.Write(req.Header.Method())
	buffer.Write(req.Header.PeekAny(fasthttp.AuthHeaderKeys))
	buffer.Write(req.URI().FullURI())
	//TODO enable configure for this feature, may filter by user or share, add/remove Authorization header to hash factor
	//fmt.Println(string(req.URI().FullURI()))
	buffer.Write(req.GetRawBody())
	str:= util.MD5digestString(buffer.Bytes())

	//buffer.Reset()
	//hashBufferPool.Put(buffer)

	return str
}

type RequestCacheGet struct {
	RequestCache
}

func (filter RequestCacheGet) Name() string {
	return "get_cache"
}

func (filter RequestCacheGet) Process(ctx *fasthttp.RequestCtx) {

	if bytes.Equal(common.FaviconPath,ctx.Request.URI().Path()){
		if global.Env().IsDebug{
			log.Tracef("skip to delegate favicon.io")
		}
		ctx.Finished()
		return
	}

	//fmt.Println(ctx.Request.Header.String())

	//TODO optimize scroll API, should always point to same IP, prefer to route to where index/shard located

	cacheable := ctx.GetFlag(common.CACHEABLE, false)

	if util.CompareStringAndBytes(ctx.Request.Header.Method(), fasthttp.MethodGet) {
		cacheable = true
	}

	url := ctx.PathStr()
	args := ctx.Request.URI().QueryArgs()

	//check special path
	switch {
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

		cacheable = true
		break
	}

	if args.Has("no_cache") {
		cacheable = false
		ctx.Request.URI().QueryArgs().Del("no_cache")
	}


	//TODO fix parameter
	patterns, ok := filter.GetStringArray("pass_patterns")
	if !ok{
		patterns=[]string{"_bulk","_cat","scroll", "scroll_id","_refresh","_cluster","_ccr","_count","_flush","_ilm","_ingest","_license","_migration","_ml","_rollup","_data_stream","_open", "_close"}
	}

	//check bypass patterns
	if ok && util.ContainsAnyInArray(url, patterns) {
		if global.Env().IsDebug {
			log.Trace("url hit bypass pattern, will not be cached, ", url)
		}
		cacheable = false
	}

	ctx.Set(common.CACHEABLE, cacheable)

	if global.Env().IsDebug {
		log.Trace("cacheable,", cacheable)
	}

	if cacheable {

		//LRU 缓存可以选择开启
		//5s 内,如果相同的 hash 出现过 2 次,则缓存起来第 3 次, 有效期 10s
		//hash->count, hash->content

		hash := filter.getHash(&ctx.Request)
		//log.Error(hash," => ",string(ctx.Request.URI().Path()))
		ctx.Set(common.CACHEHASH, hash)

		item, found := filter.GetCache(hash)

		if global.Env().IsDebug {
			log.Trace("check cache:", hash, ", found:", found)
		}

		if found {

			stats.Increment("cache", "hit")
			filter.Decode(item, &ctx.Request, &ctx.Response)
			//err:=ctx.Response.Decode(item)
			//if err!=nil{
			//	log.Debugf("error to decode response from cache %v - %v",url,err)
			//	return
			//}

			ctx.Response.Cached = true
			ctx.Response.Header.DisableNormalizing()
			ctx.Response.Header.Add("CACHED", "true")
			ctx.Response.SetDestination("cache")

			if global.Env().IsDebug {
				log.Trace("cache hit:", hash, ",", string(ctx.Request.Header.Method()), ",", string(ctx.Request.RequestURI()))
			}

			ctx.Finished()
		} else {
			stats.Increment("cache", "miss")
		}
	} else {
		stats.Increment("cache", "skip")
	}
}

type RequestCacheSet struct {
	RequestCache

	Type                   string `config:"type"` //redis,local
	TTL                    string `config:"ttl"`
	AsyncSearchTTL         string `config:"async_search_ttl"`
	generalTTLDuration     time.Duration
	asyncSearchTTLDuration time.Duration
}

func (filter RequestCacheSet) GetChaosTTLDuration() time.Duration {
	baseTTL := filter.GetTTLDuration().Milliseconds()
	randomTTL := rand.Int63n(baseTTL / 5)
	return (time.Duration(baseTTL + randomTTL)) * time.Millisecond
}

func (filter RequestCacheSet) GetTTLDuration() time.Duration {
	if filter.generalTTLDuration > 0 {
		return filter.generalTTLDuration
	}

	filter.TTL = filter.GetStringOrDefault("cache_ttl", "10s")

	if filter.TTL != "" {
		dur, err := time.ParseDuration(filter.TTL)
		if err != nil {
			dur, _ = time.ParseDuration("10s")
		}
		filter.generalTTLDuration = dur
	} else {
		filter.generalTTLDuration = time.Second * 10
	}

	return filter.generalTTLDuration
}

func (filter RequestCacheSet) GetAsyncSearchTTLDuration() time.Duration {
	if filter.asyncSearchTTLDuration > 0 {
		return filter.asyncSearchTTLDuration
	}
	filter.AsyncSearchTTL = filter.GetStringOrDefault("async_search_cache_ttl", "30m")

	if filter.AsyncSearchTTL != "" {
		dur, err := time.ParseDuration(filter.AsyncSearchTTL)
		if err != nil {
			dur, _ = time.ParseDuration("30m")
		}
		filter.asyncSearchTTLDuration = dur
	} else {
		filter.asyncSearchTTLDuration = time.Minute * 30
	}
	return filter.asyncSearchTTLDuration
}

func (filter RequestCacheSet) Name() string {
	return "set_cache"
}

func (filter RequestCacheSet) Process(ctx *fasthttp.RequestCtx) {

	cacheable := ctx.GetFlag(common.CACHEABLE, false)
	if !cacheable{
		return
	}

	hash, ok := ctx.GetString(common.CACHEHASH)
	if !ok {
		hash = filter.getHash(&ctx.Request)
	}

	ctx.Response.Header.Add("CACHE-HASH", hash)

	method :=  string(ctx.Request.Header.Method())
	url := ctx.PathStr()
	//args := ctx.Request.URI().QueryArgs()

	//cache 200 only TODO allow configure to support: 404/200/201/500, also set TTL
	if ctx.Response.StatusCode() == http.StatusOK {

		//TODO check max_response_size, skip if the response is too big

		body := ctx.Response.Body()
		var id string

		//buffer := bytesBufferPool.Get().(*bytes.Buffer)

		cacheBytes := filter.Encode(ctx)
		//cacheBytes := ctx.Response.Encode()

		//buffer.Reset()
		//bytesBufferPool.Put(buffer)


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
				//if response contains:
				//"id" : "FktyZXA2bklVU2VDeWIwVWdkVTlMcGcdMWpuRkM3SDZSWWVBSTdKT1hkRDNkdzoyNDY3MjY=",
				//then it is a async task, store ID to cache, and if this task finished, associate that result to this same request
				if ok {
					if global.Env().IsDebug {
						log.Trace("async hash: set async hash cache", string(id), "=>", string(hash))
					}
					filter.SetCache(id, []byte(hash), filter.GetAsyncSearchTTLDuration())
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
						item, found := filter.GetCache(id)
						if found {
							if global.Env().IsDebug {
								log.Trace("found request hash, set cache:", id, ": ", string(item))
							}
							filter.SetCache(string(item), cacheBytes, filter.GetChaosTTLDuration())
						} else {
							if global.Env().IsDebug {
								log.Trace("async search request hash was lost:", id)
							}
						}
					}
				}
			}
		}


		filter.SetCache(hash, cacheBytes, filter.GetChaosTTLDuration())
		if global.Env().IsDebug {
			log.Trace("cache was stored")
		}
	}
}



//TODO optimize memmove issue, buffer read
func (p RequestCache) Decode(data []byte, req *fasthttp.Request, res *fasthttp.Response) {
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

func (p RequestCache) Encode(ctx *fasthttp.RequestCtx)[]byte {

	buffer := bytes.Buffer{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		buffer.Write(key)
		buffer.Write(colon)
		buffer.Write(value)
		buffer.Write(newLine)
	})

	header := buffer.Bytes()
	body := ctx.Response.Body()

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

