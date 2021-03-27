package transform

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"regexp"
)

type RequestBodyRegexReplace struct {
	param.Parameters
}

func (filter RequestBodyRegexReplace) Name() string {
	return "request_body_regex_replace"
}

func (filter RequestBodyRegexReplace) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
	pattern:=filter.MustGetString("pattern")
	to:=filter.MustGetString("to")

	if global.Env().IsDebug{
		log.Trace("pattern:",pattern,", to:",to)
	}

	body:=ctx.Request.GetRawBody()
	if len(body)>0{
		p,err:=regexp.Compile(pattern)
		if err!=nil{
			log.Error(err)
			return
		}

		newBody:=p.ReplaceAll(body,[]byte(to))
		//log.Error("new body:")
		//log.Error(string(newBody))

		//TODO auto handle uncompressed response, gzip after all
		ctx.Request.Header.Del(fasthttp.HeaderContentEncoding)
		ctx.Request.Header.Del(fasthttp.HeaderContentEncoding2)
		ctx.Request.SetBody(newBody)
	}
}
