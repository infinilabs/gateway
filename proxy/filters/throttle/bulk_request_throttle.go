package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

type ElasticsearchBulkRequestThrottle struct {
	Indices         map[string]*config.Config `config:"indices"`
	hashWildcard    bool
	indicesLimiter  map[string]*GenericLimiter
	indicesPatterns map[string]*radix.Pattern
}

func (filter *ElasticsearchBulkRequestThrottle) Name() string {
	return "bulk_request_throttle"
}

func (this *ElasticsearchBulkRequestThrottle) Filter(ctx *fasthttp.RequestCtx) {

	if len(this.Indices) < 0 {
		log.Warn("no indices was configured")
		return
	}

	pathStr := util.UnsafeBytesToString(ctx.URI().Path())

	if util.SuffixStr(pathStr, "/_bulk") {

		body := ctx.Request.GetRawBody()
		var indexOpStats = map[string]int{}
		var indexPayloadStats = map[string]int{}

		docCount, err := elastic.WalkBulkRequests(body, func(eachLine []byte) (skipNextLine bool) {
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id,routing string) (err error) {
			if index==""{
				//url level
				var urlLevelIndex string
				urlLevelIndex, _ = elastic.ParseUrlLevelBulkMeta(pathStr)
				if urlLevelIndex != "" {
					index = urlLevelIndex
				}
			}

			//stats
			v, ok := indexOpStats[index]
			if !ok {
				indexOpStats[index] = 1
			} else {
				indexOpStats[index] = v + 1
			}

			v, ok = indexPayloadStats[index]
			if !ok {
				indexPayloadStats[index] = len(metaBytes)
			} else {
				indexPayloadStats[index] = v + len(metaBytes)
			}
			return nil
		}, func(payloadBytes []byte, actionStr, index, typeName, id,routing string) {
			v, ok := indexPayloadStats[index]
			if !ok {
				indexPayloadStats[index] = len(payloadBytes)
			} else {
				indexPayloadStats[index] = v + len(payloadBytes)
			}
		})

		if global.Env().IsDebug{
			log.Debug(indexOpStats)
			log.Debug(indexPayloadStats)
		}

		for k,hits:=range indexOpStats{
			bytes,ok1:=indexPayloadStats[k]
			if !ok1{
				continue
			}
			limiter,ok:=this.indicesLimiter[k]
			if global.Env().IsDebug {
				log.Debug("index:",k, " met bulk check rules, hits:",hits,",bytes:",bytes)
			}
			if !ok{
				if this.hashWildcard{
					for x,y:=range this.indicesPatterns{
						ok := y.Match(k)
						if global.Env().IsDebug{
							log.Trace("hit index pattern:",x,",",k)
						}
						if ok{
							limiter,ok=this.indicesLimiter[x]
							//TODO may support multi-patterns
							break
						}
					}
				}
			}
			if limiter!=nil{
				limiter.internalProcessWithValues("bulk_requests", k, ctx,hits,bytes)
			}
		}


		if err != nil {
			log.Errorf("processing: %v docs, err: %v", docCount, err)
			return
		}

	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_request_throttle", NewElasticsearchBulkRequestThrottleFilter, &ElasticsearchBulkRequestThrottle{})
}

func NewElasticsearchBulkRequestThrottleFilter(c *config.Config) (pipeline.Filter, error) {
	runner := ElasticsearchBulkRequestThrottle{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.indicesPatterns= map[string]*radix.Pattern{}
	runner.indicesLimiter= map[string]*GenericLimiter{}

	for k,v:=range runner.Indices{
		if strings.Contains(k,"*"){
			runner.hashWildcard=true
			patterns := radix.Compile(k)
			runner.indicesPatterns[k]=patterns
		}

		limiter := genericLimiter
		if err := v.Unpack(&limiter); err != nil {
			return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
		}
		limiter.init()
		runner.indicesLimiter[k]=&limiter
	}

	if global.Env().IsDebug{
		log.Trace(util.ToJson(runner.indicesLimiter,true))
	}

	return &runner, nil
}
