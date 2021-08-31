package throttle

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type SleepFilter struct {
	param.Parameters
}

func (filter SleepFilter) Name() string {
	return "sleep"
}

func (filter SleepFilter) Process(ctx *fasthttp.RequestCtx) {
	sleepInMs,ok:=filter.GetInt64("sleep_in_million_seconds",-1)
	if !ok{
		return
	}
	time.Sleep(time.Duration(sleepInMs)*time.Millisecond)
}

type DropFilter struct {
	param.Parameters
}

func (filter DropFilter) Name() string {
	return "drop"
}

func (filter DropFilter) Process(ctx *fasthttp.RequestCtx) {
	ctx.Finished()
}

type ElasticsearchHealthCheckFilter struct {
	param.Parameters
}

func (filter ElasticsearchHealthCheckFilter) Name() string {
	return "elasticsearch_health_check"
}

func (filter ElasticsearchHealthCheckFilter) Process(ctx *fasthttp.RequestCtx) {
	esName:=filter.MustGetString("elasticsearch")
	if rate.GetRateLimiter("cluster_check_health",esName,1,1,time.Second*1).Allow(){
		result:=elastic.GetClient(esName).ClusterHealth()
		if global.Env().IsDebug{
			log.Trace(esName,result)
		}
		if result.StatusCode==200||result.StatusCode==403{
			cfg:=elastic.GetMetadata(esName)
			if cfg!=nil{
				cfg.ReportSuccess()
			}
		}
	}
}
