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

type ResponseBodyRegexReplace struct {
	Pattern string `config:"pattern"`
	To      string `config:"to"`
	p       *regexp.Regexp
}

func (filter *ResponseBodyRegexReplace) Name() string {
	return "response_body_regex_replace"
}

func (filter *ResponseBodyRegexReplace) Filter(ctx *fasthttp.RequestCtx) {
	if global.Env().IsDebug {
		log.Trace("pattern:", filter.Pattern, ", to:", filter.To)
	}

	//c,v:=ctx.Response.IsCompressed()
	//log.Error(c," || ",string(v))

	body := ctx.Response.GetRawBody()
	if len(body) > 0 {

		newBody := filter.p.ReplaceAll(body, util.UnsafeStringToBytes(filter.To))

		//TODO auto handle uncompressed response
		ctx.Response.Header.Del(fasthttp.HeaderContentEncoding)
		ctx.Response.Header.Del(fasthttp.HeaderContentEncoding2)
		ctx.Response.SetBody(newBody)
	}
}

func NewResponseBodyRegexReplace(c *config.Config) (filter pipeline.Filter, err error) {

	runner := ResponseBodyRegexReplace{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.p, err = regexp.Compile(runner.Pattern)
	if err != nil {
		panic(err)
	}
	return &runner, nil
}
