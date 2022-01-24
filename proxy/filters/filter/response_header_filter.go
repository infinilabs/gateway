package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseHeaderFilter struct {
	genericFilter *RequestFilter
	Include       []map[string]string `config:"include"`
	Exclude       []map[string]string `config:"exclude"`
}

func (filter *ResponseHeaderFilter) Name() string {
	return "response_header_filter"
}

func init() {
	pipeline.RegisterFilterPlugin("response_header_filter",NewResponseHeaderFilter)
}

func NewResponseHeaderFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ResponseHeaderFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.genericFilter = &RequestFilter{
		Action: "deny",
		Status: 403,
	}

	if err := c.Unpack(runner.genericFilter); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *ResponseHeaderFilter) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Debug("headers:", string(util.EscapeNewLine(ctx.Response.Header.Header())))
	}

	if len(filter.Exclude) > 0 {
		for _, x := range filter.Exclude {
			for k, v := range x {
				v1 := ctx.Response.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("exclude header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					filter.genericFilter.Filter(ctx)
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
	}

	if len(filter.Include) > 0 {
		for _, x := range filter.Include {
			for k, v := range x {
				v1 := ctx.Response.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("include header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
		filter.genericFilter.Filter(ctx)
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}

}
