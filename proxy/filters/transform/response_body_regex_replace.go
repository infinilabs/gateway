package transform

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"regexp"
)

type ResponseBodyRegexReplace struct {
	param.Parameters
}

func (filter ResponseBodyRegexReplace) Name() string {
	return "response_body_regex_replace"
}

func (filter ResponseBodyRegexReplace) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
	pattern:=filter.MustGetString("pattern")
	to:=filter.MustGetString("to")

	if global.Env().IsDebug{
		log.Trace("pattern:",pattern,", to:",to)
	}

	//c,v:=ctx.Response.IsCompressed()
	//log.Error(c," || ",string(v))

	body:=ctx.Response.GetRawBody()
	if len(body)>0{
		//log.Error("old body:")
		//log.Error(string(body))
		p,err:=regexp.Compile(pattern)
		if err!=nil{
			log.Error(err)
			return
		}

		newBody:=p.ReplaceAll(body,[]byte(to))
		//log.Error("new body:")
		//log.Error(string(newBody))

		//TODO auto handle uncompressed response
		ctx.Response.Header.Del(fasthttp.HeaderContentEncoding)
		ctx.Response.Header.Del(fasthttp.HeaderContentEncoding2)
		ctx.Response.SetBody(newBody)
	}
}
