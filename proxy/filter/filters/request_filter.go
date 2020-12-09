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
						log.Debugf("rule matched, this request has been filtered: %v",ctx.Request.URI().String())
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
					if global.Env().IsDebug{
						log.Debugf("rule matched, this request has been marked as good one: %v",ctx.Request.URI().String())
					}
					return
				}
			}
		}
		ctx.Finished()
		if global.Env().IsDebug{
			log.Debugf("no rule matched, this request has been filtered: %v",ctx.Request.URI().String())
		}
	}

}


type RequestMethodFilter struct {
	RequestFilter
}


func (filter RequestMethodFilter) Name() string {
	return "request_method_filter"
}

func (filter RequestMethodFilter) Process(ctx *fasthttp.RequestCtx) {

	method:=string(ctx.Method())

	if global.Env().IsDebug{
		log.Debug("method:",method)
	}

	exclude,ok:=filter.GetStringArray("exclude")
	if ok{
		for _,x:=range exclude{
				if global.Env().IsDebug{
					log.Debugf("exclude method: %v vs %v, match: %v",x,method,util.ToString(x)==method)
				}
				if util.ToString(x)==method{
					ctx.Finished()
					if global.Env().IsDebug{
						log.Debugf("rule matched, this request has been filtered: %v",ctx.Request.URI().String())
					}
					return
				}
		}
	}

	include,ok:=filter.GetStringArray("include")
	if ok{
		for _,x:=range include{
				if global.Env().IsDebug{
					log.Debugf("include method [%v]: %v vs %v, match: %v",x,method,util.ToString(x)==string(method))
				}
				if util.ToString(x)== method {
					if global.Env().IsDebug{
						log.Debugf("rule matched, this request has been marked as good one: %v",ctx.Request.URI().String())
					}
					return
				}
		}
		ctx.Finished()
		if global.Env().IsDebug{
			log.Debugf("no rule matched, this request has been filtered: %v",ctx.Request.URI().String())
		}
	}

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
