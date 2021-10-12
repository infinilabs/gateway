package deprecated

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type RequestBodyTruncate struct {
	param.Parameters
}

func (filter RequestBodyTruncate) Name() string {
	return "request_body_truncate"
}

func (filter RequestBodyTruncate) Filter(ctx *fasthttp.RequestCtx) {
	size:=filter.GetIntOrDefault("max_size",1024)
	addHeader:=filter.GetBool("add_header",true)

	if global.Env().IsDebug{
		log.Trace("limit:",size,", actual:",ctx.Request.GetBodyLength())
	}
	if ctx.Request.GetBodyLength()>size{
		if addHeader{
			headerMessage:=fmt.Sprintf("%v -> %v",ctx.Request.GetBodyLength(),size)
			ctx.Request.Header.Set("REQUEST_BODY_TRUNCATED",headerMessage)
		}
		ctx.Request.ResetBodyLength(size)
	}
}
