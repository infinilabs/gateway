package transform

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type RequestBodyRegexReplace struct {
	Pattern string `config:"pattern"`
	To string `config:"to"`
	p *regexp.Regexp
}

func (filter *RequestBodyRegexReplace) Name() string {
	return "request_body_regex_replace"
}

func (filter *RequestBodyRegexReplace) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug{
		log.Trace("pattern:",filter.Pattern,", to:",filter.To)
	}

	body:=ctx.Request.GetRawBody()
	if len(body)>0{
		newBody:=filter.p.ReplaceAll(body,util.UnsafeStringToBytes(filter.To))
		ctx.Request.SetRawBody(newBody)
	}
}

func NewRequestBodyRegexReplace(c *config.Config) (filter pipeline.Filter, err error) {

	runner := RequestBodyRegexReplace{
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.p,err=regexp.Compile(runner.Pattern)
	if err!=nil{
		panic(err)
	}
	return &runner, nil
}
