package elastic

import (
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"sync"
	"time"
)

type Elasticsearch struct {
	param.Parameters
}

func (filter Elasticsearch) Name() string {
	return "elasticsearch"
}

var proxyList = sync.Map{}

var initLock sync.Mutex

var faviconPath=[]byte("/favicon.ico")

//var singleSetCache singleflight.Group

func (filter Elasticsearch) Process(ctx *fasthttp.RequestCtx) {

	if bytes.Equal(faviconPath,ctx.Request.URI().Path()){
		if global.Env().IsDebug{
			log.Tracef("skip to delegate favicon.io")
		}
		ctx.Finished()
		return
	}

	var instance *ReverseProxy

	esRef := filter.MustGetString("elasticsearch")

	cfg:=elastic.GetConfig(esRef)

	instance1, ok := proxyList.Load(esRef)
	if instance1!=nil{
		instance=instance1.(*ReverseProxy)
	}

	if !ok || instance == nil {

		var proxyConfig = ProxyConfig{}
		proxyConfig.Elasticsearch = esRef
		proxyConfig.Balancer = filter.GetStringOrDefault("balancer", "weight")
		proxyConfig.MaxResponseBodySize = filter.GetIntOrDefault("max_response_size", 100*1024*1024)
		proxyConfig.MaxConnection = filter.GetIntOrDefault("max_connection", 10000)
		proxyConfig.maxRetryTimes = filter.GetIntOrDefault("max_retry_times", 10)
		proxyConfig.retryDelayInMs = filter.GetIntOrDefault("retry_delay_in_ms", 1000)
		proxyConfig.TLSInsecureSkipVerify = filter.GetBool("tls_insecure_skip_verify", true)

		proxyConfig.ReadBufferSize = filter.GetIntOrDefault("read_buffer_size", 4096*4)
		proxyConfig.WriteBufferSize = filter.GetIntOrDefault("write_buffer_size", 4096*4)

		proxyConfig.MaxConnWaitTimeout = filter.GetDurationOrDefault("max_conn_wait_timeout", "0s")
		proxyConfig.MaxIdleConnDuration = filter.GetDurationOrDefault("max_idle_conn_duration", "10s")
		proxyConfig.MaxConnDuration = filter.GetDurationOrDefault("max_conn_duration", "0s")
		proxyConfig.ReadTimeout = filter.GetDurationOrDefault("read_timeout", "0s")
		proxyConfig.WriteTimeout = filter.GetDurationOrDefault("write_timeout", "0s")

		if filter.Has("filter") {
			filter.Config("filter", &proxyConfig.Filter)
		}

		if filter.Has("refresh") {
			filter.Config("refresh", &proxyConfig.Refresh)
		}

		instance = NewReverseProxy(&proxyConfig)
		proxyList.Store(esRef,instance)
	}

	if !cfg.IsAvailable(){
		//log.Error(fmt.Sprintf("cluster [%v] is not available",esRef))

		if rate.GetRateLimiter("cluster_check_health",cfg.Name,1,1,time.Second*1).Allow(){
			result:=elastic.GetClient(cfg.Name).ClusterHealth()
			if result.StatusCode==200{
				cfg.ReportSuccess()
			}
		}

		ctx.Response.SwapBody([]byte(fmt.Sprintf("cluster [%v] is not available",esRef)))
		ctx.SetStatusCode(500)
		ctx.Finished()
		time.Sleep(1*time.Second)
		return
	}

	instance.DelegateRequest(esRef,cfg,ctx)
}
