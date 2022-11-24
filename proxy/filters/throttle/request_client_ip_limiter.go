package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	config "infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RequestClientIPLimitFilter struct {
	limiter *GenericLimiter
	IP      []string `config:"ip"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_client_ip_limiter",NewRequestClientIPLimitFilter,&RequestClientIPLimitFilter{})
}

func NewRequestClientIPLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestHostLimitFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	limiter := genericLimiter
	runner.limiter = &limiter

	if err := c.Unpack(runner.limiter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.limiter.init()

	return &runner, nil
}

func (filter *RequestClientIPLimitFilter) Name() string {
	return "request_client_ip_limiter"
}

func (filter *RequestClientIPLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	clientIP := ctx.RemoteIP().String()

	if global.Env().IsDebug {
		log.Trace("ips rules: ", len(filter.IP), ", client_ip: ", clientIP)
	}

	if len(filter.IP) > 0 {
		for _, v := range filter.IP {
			//check if ip pre-defined
			if v == clientIP {
				if global.Env().IsDebug {
					log.Debug(clientIP, "met check rules")
				}
				filter.limiter.internalProcess("client_ip", clientIP, ctx)
				return
			}
		}
		//terminate if no ip matches
		return
	}

	filter.limiter.internalProcess("client_ip", clientIP, ctx)
}
