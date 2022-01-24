package transform

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseHeaderFormatFilter struct {
}

func (filter *ResponseHeaderFormatFilter) Name() string {
	return "response_header_format"
}

func (filter *ResponseHeaderFormatFilter) Filter(ctx *fasthttp.RequestCtx) {

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.SetBytesKV(util.ToLowercase(key), value)
	})
}

func init() {
	pipeline.RegisterFilterPlugin("response_header_format",NewResponseHeaderFormatFilter)
}

func NewResponseHeaderFormatFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ResponseHeaderFormatFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
