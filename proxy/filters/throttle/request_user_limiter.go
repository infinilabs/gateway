package throttle


import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type RequestUserLimitFilter struct {
	RequestLimiterBase
}

func (filter RequestUserLimitFilter) Name() string {
	return "request_user_limiter"
}

func (filter RequestUserLimitFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	exists,user,_:=ctx.ParseBasicAuth()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("user not exist")
		}
		return
	}

	ips, ok := filter.GetStringArray("user")

	userStr:=string(user)
	if global.Env().IsDebug {
		log.Trace("user rules: ", len(ips), ", user: ", userStr)
	}

	if ok&&len(ips) > 0 {
		for _, v := range ips {
			if v == userStr {
				if global.Env().IsDebug {
					log.Debug(userStr, "met check rules")
				}
				filter.internalProcess("user",userStr,filterCfg,ctx)
				return
			}
		}
		//terminate if no ip matches
		return
	}

	filter.internalProcess("user",userStr,filterCfg,ctx)
}
