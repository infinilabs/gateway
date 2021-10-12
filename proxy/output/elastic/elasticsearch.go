package elastic

import (
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type Elasticsearch struct {
	param.Parameters
	config *ProxyConfig
	instance *ReverseProxy
}

func (filter *Elasticsearch) Name() string {
	return "elasticsearch"
}

var faviconPath=[]byte("/favicon.ico")

//var singleSetCache singleflight.Group

func (filter *Elasticsearch) Filter(ctx *fasthttp.RequestCtx) {

	if bytes.Equal(faviconPath,ctx.Request.URI().Path()){
		if global.Env().IsDebug{
			log.Tracef("skip to delegate favicon.io")
		}
		ctx.Finished()
		return
	}


	metadata :=elastic.GetMetadata(filter.config.Elasticsearch)

	if metadata !=nil&&!metadata.IsAvailable(){
		if rate.GetRateLimiter("cluster_check_health", metadata.Config.ID,1,1,time.Second*1).Allow(){
			log.Error(fmt.Sprintf("cluster [%v] is not available",filter.config.Elasticsearch))
			result:=elastic.GetClient(metadata.Config.Name).ClusterHealth()
			if result.StatusCode==200{
				metadata.ReportSuccess()
			}
		}

		ctx.Response.SwapBody([]byte(fmt.Sprintf("cluster [%v] is not available",filter.config.Elasticsearch)))
		ctx.SetStatusCode(500)
		ctx.Finished()
		log.Infof("cluster [%v] not available",filter.config.Elasticsearch)
		time.Sleep(1*time.Second)
		return
	}

	//TODO move clients selection async

	filter.instance.DelegateRequest(filter.config.Elasticsearch, metadata,ctx)
}


func New(c *config.Config) (pipeline.Filter, error) {

	cfg := ProxyConfig{
		Balancer: "weight",
		MaxResponseBodySize: 100*1024*1024,
		MaxConnection: 5000,
		maxRetryTimes: 10,
		retryDelayInMs: 1000,
		TLSInsecureSkipVerify: true,
		ReadBufferSize: 4096*4,
		WriteBufferSize: 4096*4,
		MaxConnWaitTimeout: util.GetDurationOrDefault("10s",10*time.Second),
		MaxConnDuration: util.GetDurationOrDefault("0s",10*time.Second),
		ReadTimeout: util.GetDurationOrDefault("0s",10*time.Second),
		WriteTimeout: util.GetDurationOrDefault("0s",10*time.Second),

	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := Elasticsearch{config: &cfg}

	runner.instance = NewReverseProxy(&cfg)

	log.Debugf("init elasticsearch proxy instance: %v",cfg.Elasticsearch )

	return &runner, nil
}