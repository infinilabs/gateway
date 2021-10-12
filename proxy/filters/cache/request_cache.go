package cache

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type RequestCache struct {
	config *Config
}


type Config struct {
	CacheType string    `config:"cache_type"`
	PassPatterns []string    `config:"pass_patterns"`
	ValidatedStatus []int    `config:"validated_status_code"`
	MinResponseSize int    `config:"min_response_size"`
	MaxResponseSize int    `config:"max_response_size"`

	RedisHost string    `config:"redis_host"`
	RedisPort int    `config:"redis_port"`


	MaxCachedSize int64    `config:"max_cached_size"`
	MaxCachedItem int64    `config:"max_cached_item"`

	AsyncSearchCacheTTL string    `config:"async_search_cache_ttl"`
	CacheTTL string    `config:"cache_ttl"`
	asyncSearchCacheTTL time.Duration
	cacheTTL time.Duration

}

var defaultConfig=Config{
	PassPatterns:[]string{"_bulk","_cat","scroll", "scroll_id","_refresh","_cluster","_ccr","_count","_flush","_ilm","_ingest","_license","_migration","_ml","_rollup","_data_stream","_open", "_close"},
	ValidatedStatus:[]int{200,201,404,403,413,400,301},
	AsyncSearchCacheTTL:"30m",
	MinResponseSize:-1,
	MaxResponseSize:int(^uint(0) >> 1),
	MaxCachedSize:1000000000,
	MaxCachedItem:1000000,
	CacheType:defaultCacheType,
}

func NewGet(c *config.Config) (pipeline.Filter, error) {

	cfg := defaultConfig

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	cfg.asyncSearchCacheTTL=util.GetDurationOrDefault(cfg.AsyncSearchCacheTTL,10*time.Minute)
	cfg.cacheTTL=util.GetDurationOrDefault(cfg.CacheTTL,10*time.Second)

	runner := RequestCacheGet{config: &cfg}
	runner.RequestCache.config=&cfg

	runner.initCache()

	return &runner, nil
}

func NewSet(c *config.Config) (pipeline.Filter, error) {

	cfg := defaultConfig

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	cfg.asyncSearchCacheTTL=util.GetDurationOrDefault(cfg.AsyncSearchCacheTTL,10*time.Minute)
	cfg.cacheTTL=util.GetDurationOrDefault(cfg.CacheTTL,10*time.Second)

	runner := RequestCacheSet{config: &cfg}
	runner.RequestCache.config=&cfg

	runner.initCache()

	return &runner, nil
}

const cacheRedis = "redis"
const cacheCCache = "ccache"
const ristrettoCache = "ristretto"
const defaultCacheType = "ristretto"

var ccCache *ccache.LayeredCache
var l sync.RWMutex
var colon = []byte(": ")
var newLine = []byte("\n")
var client *redis.Client
var cache *ristretto.Cache
var inited bool
var ctx = context.Background()

var bytesBufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (p *RequestCache) getRedisClient() *redis.Client {

	if client != nil {
		return client
	}

	l.Lock()
	defer l.Unlock()

	if client != nil {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%v", p.config.RedisHost, p.config.RedisPort),
		//Password: handler.config.RedisConfig.Password,
		//DB:       handler.config.RedisConfig.DB,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}

	return client
}

func (p *RequestCache) initCache() {
	if inited {
		return
	}

	l.Lock()
	defer l.Unlock()

	if inited {
		return
	}

	var err error
	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,                                                // Num keys to track frequency of (10M).
		MaxCost:     p.config.MaxCachedSize, 							 // Maximum cost of cache (1GB).
		BufferItems: 64,                                                 // Number of keys per Get buffer.
		Metrics:     false,
	})
	if err != nil {
		panic(err)
	}

	ccCache = ccache.Layered(ccache.Configure().MaxSize(p.config.MaxCachedItem).ItemsToPrune(100))
	inited = true
}

func (p *RequestCache) GetCache(key string) ([]byte, bool) {
	item := ccCache.GetOrCreateSecondaryCache("default").Get(key)
	if item != nil {
		data := item.Value().([]byte)
		if item.Expired() {
			stats.Increment("cache", "expired")
			ccCache.GetOrCreateSecondaryCache("default").Delete(key)
		}
		return data, true
	}
	switch p.config.CacheType {
		case cacheRedis:
			b, err := p.getRedisClient().Get(ctx,key).Result()
			if err == redis.Nil {
				return nil, false
			} else if err != nil {
				return nil, false
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
				return o.([]byte), true
			}
		}
	return nil, false
}

func (p *RequestCache) SetCache(key string, data []byte, ttl time.Duration) {

	if global.Env().IsDebug {
		log.Trace("set cache:", key, ", ttl:", ttl)
	}

	dataLen := len(data)
	if dataLen < p.config.MinResponseSize || (dataLen > p.config.MaxResponseSize) && p.config.MaxResponseSize>0 {
		if global.Env().IsDebug {
			log.Tracef("invalid response size, %v not between %v and %v", dataLen, p.config.MinResponseSize, p.config.MaxResponseSize)
		}
		return
	}

	switch p.config.CacheType {
	case cacheRedis:
		err := p.getRedisClient().Set(ctx,key, data, ttl).Err()
		if err != nil {
			panic(err)
		}
		return
	case cacheCCache:
		ccCache.GetOrCreateSecondaryCache("default").Set(key, data, ttl)
		return
	default:
		cache.SetWithTTL(key, data, int64(dataLen), ttl)
	}
}

func (p *RequestCache) getHash(ctx *fasthttp.RequestCtx) string {

	//TODO configure, remove keys from hash factor
	ctx.Request.URI().QueryArgs().Del("preference")

	buffer:=bytes.Buffer{}
	//buffer:=hashBufferPool.Get().(*bytes.Buffer)

	if global.Env().IsDebug {
		log.Trace("generate hash:", string(ctx.Request.Header.Method()), string(ctx.Request.RequestURI()), string(ctx.Request.URI().QueryArgs().QueryString()), string(ctx.Request.Body()), string(ctx.Request.PostArgs().QueryString()))
	}

	//TODO 后台可以按照请求路径来勾选 Hash 因子

	buffer.Write(ctx.Request.Header.Method())
	//TODO enable configure for this feature, may filter by user or share, add/remove Authorization header to hash factor
	buffer.Write(ctx.Request.Header.PeekAny(fasthttp.AuthHeaderKeys))
	buffer.Write(ctx.Request.URI().FullURI())
	buffer.Write(ctx.Request.GetRawBody())
	str:= util.MD5digestString(buffer.Bytes())

	//buffer.Reset()
	//hashBufferPool.Put(buffer)

	return str
}

type RequestCacheGet struct {
	RequestCache
	config *Config
}

func (filter *RequestCacheGet) Name() string {
	return "get_cache"
}

func (filter *RequestCacheGet) Filter(ctx *fasthttp.RequestCtx) {
	if bytes.Equal(common.FaviconPath,ctx.Request.URI().Path()){
		if global.Env().IsDebug{
			log.Tracef("skip to delegate favicon.io")
		}
		ctx.Finished()
		return
	}

	//TODO optimize scroll API, should always point to same IP, prefer to route to where index/shard located

	var cacheable = false

	if util.CompareStringAndBytes(ctx.Request.Header.Method(), fasthttp.MethodGet) {
		cacheable = true
	}

	url := string(ctx.RequestURI())
	args := ctx.Request.URI().QueryArgs()

	//check special path
	switch {
	case util.ContainStr(url, "/_search"):
		cacheable = true
		break
	case util.ContainsAnyInArray(url, []string{"_mget", "/_security/user/_has_privileges"}):
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

	//check bypass patterns
	if len(filter.config.PassPatterns)>0&&util.ContainsAnyInArray(url, filter.config.PassPatterns) {
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

		hash := filter.getHash(ctx)
		ctx.Set(common.CACHEHASH, hash)

		item, found := filter.GetCache(hash)

		if global.Env().IsDebug {
			log.Trace("check cache:", hash, ", found:", found)
		}

		if found {

			stats.Increment("cache", "hit")
			err:=ctx.Response.Decode(item)
			if err!=nil{
				log.Error(err)
				panic(err)
			}
			ctx.Response.Cached = true
			ctx.Response.Header.Add("CACHED", "true")
			ctx.Response.Header.Add("CACHE-HASH", hash)
			ctx.SetDestination("cache")

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
	config *Config
	Type                   string `config:"type"` //redis,local
	TTL                    string `config:"ttl"`
	AsyncSearchTTL         string `config:"async_search_ttl"`
	generalTTLDuration     time.Duration
	asyncSearchTTLDuration time.Duration
}

func (filter *RequestCacheSet) GetChaosTTLDuration() time.Duration {
	baseTTL := filter.config.cacheTTL.Milliseconds()
	randomTTL := rand.Int63n(baseTTL / 5)
	return (time.Duration(baseTTL + randomTTL)) * time.Millisecond
}


func (filter *RequestCacheSet) Name() string {
	return "set_cache"
}

func (filter *RequestCacheSet) Filter(ctx *fasthttp.RequestCtx) {
	method :=  string(ctx.Request.Header.Method())
	url := string(ctx.RequestURI())

	cacheable := ctx.GetBool(common.CACHEABLE, false)
	if !cacheable{
		if global.Env().IsDebug{
			log.Trace("not cacheable ",cacheable,",",url)
		}
		return
	}

	hash, ok := ctx.GetString(common.CACHEHASH)

	if !ok {
		hash = filter.getHash(ctx)
	}

	if util.ContainsInAnyInt32Array(ctx.Response.StatusCode(),filter.config.ValidatedStatus){

		body := ctx.Response.Body()

		//check max_response_size, skip if the response is too big
		if ok{
			if filter.config.MaxResponseSize>0 && len(body)>filter.config.MaxResponseSize{
				log.Warnf("response is too big ( %v > %v ), skip to put into cache",len(body),filter.config.MaxResponseSize)
				return
			}
		}

		var id string

		cacheBytes := ctx.Response.Encode()

		if len(cacheBytes)==0{
			log.Warn("invalid cache bytes")
			return
		}

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
					filter.SetCache(id, []byte(hash), filter.config.asyncSearchCacheTTL)
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


