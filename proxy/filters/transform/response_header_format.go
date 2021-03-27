package transform

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type ResponseHeaderFormatFilter struct {
	param.Parameters
}

func (filter ResponseHeaderFormatFilter) Name() string {
	return "response_header_format"
}

func (filter ResponseHeaderFormatFilter) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.SetBytesKV(util.ToLowercase(key),value)
	})

}

