package deprecated

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type ResponseBodyTruncate struct {
	param.Parameters
}

func (filter ResponseBodyTruncate) Name() string {
	return "response_body_truncate"
}

func (filter ResponseBodyTruncate) Filter(ctx *fasthttp.RequestCtx) {
	size:=filter.GetIntOrDefault("max_size",1024)
	addHeader:=filter.GetBool("add_header",true)

	if global.Env().IsDebug{
		log.Trace("limit:",size,", actual:",ctx.Response.GetBodyLength())
	}
	if ctx.Response.GetBodyLength()>size{
		if addHeader{
			headerMessage:=fmt.Sprintf("%v -> %v",ctx.Response.GetBodyLength(),size)
			ctx.Response.Header.Set("RESPONSE_BODY_TRUNCATED",headerMessage)
		}
		ctx.Response.ResetBodyLength(size)
	}
}
