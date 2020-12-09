package filters

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	log "github.com/cihub/seelog"
)

type RequestFilter struct {
	param.Parameters
}

type RequestHeaderFilter struct {
	RequestFilter
}


func (filter RequestHeaderFilter) Name() string {
	return "request_header_filter"
}

func (filter RequestHeaderFilter) Process(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug{
		log.Debug("headers:",string(util.EscapeNewLine(ctx.Request.Header.Header())))
	}

	exclude,ok:=filter.GetMapArray("exclude")
	if ok{
		for _,x:=range exclude{
			for k,v:=range x{
				v1:=ctx.Request.Header.Peek(k)
				if global.Env().IsDebug{
					log.Debugf("exclude header [%v]: %v vs %v, match: %v",k,v,string(v1),util.ToString(v)==string(v1))
				}
				if util.ToString(v)==string(v1){
					ctx.Finished()
					if global.Env().IsDebug{
						log.Debugf("this request has been filtered: %v",ctx.Request.URI().String())
					}
					return
				}
			}
		}
	}

	include,ok:=filter.GetMapArray("include")
	if ok{
		for _,x:=range include{
			for k,v:=range x{
				v1:=ctx.Request.Header.Peek(k)
				if global.Env().IsDebug{
					log.Debugf("include header [%v]: %v vs %v, match: %v",k,v,string(v1),util.ToString(v)==string(v1))
				}
				if util.ToString(v)==string(v1){
					return
				}
			}
		}
		ctx.Finished()
		if global.Env().IsDebug{
			log.Debugf("this request has been filtered: %v",ctx.Request.URI().String())
		}
	}

}


type RequestMethodFilter struct {
	RequestFilter
}

type RequestUrlPathFilter struct {
	RequestFilter
}

type RequestUrlQueryArgsFilter struct {
	RequestFilter
}

type RequestBodyFilter struct {
	RequestFilter
}

type ResponseCodeFilter struct {
	RequestFilter
}

type ResponseHeaderFilter struct {
	RequestFilter
}

type ResponseBodyFilter struct {
	RequestFilter
}
