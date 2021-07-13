package transform

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type SetRequestHeader struct {
	param.Parameters
}

func (filter SetRequestHeader) Name() string {
	return "set_request_header"
}

func (filter SetRequestHeader) Process(ctx *fasthttp.RequestCtx) {
	headers,ok := filter.GetStringMap("headers")

	if !ok{
		return
	}

	for k,v:=range headers{
		//remove old one
		value:=ctx.Request.Header.Peek(k)
		if len(value)>0{
			ctx.Request.Header.Del(k)
		}
		ctx.Request.Header.Set(k,v)
	}
}

type SetResponseHeader struct {
	param.Parameters
}

func (filter SetResponseHeader) Name() string {
	return "set_response_header"
}

func (filter SetResponseHeader) Process(ctx *fasthttp.RequestCtx) {
	headers,ok := filter.GetStringMap("headers")

	if !ok{
		return
	}

	for k,v:=range headers{
		//remove old one
		value:=ctx.Request.Header.Peek(k)
		if len(value)>0{
			ctx.Request.Header.Del(k)
		}
		ctx.Request.Header.Set(k,v)
	}
}

type SetHostname struct {
	param.Parameters
}

func (filter SetHostname) Name() string {
	return "set_hostname"
}

func (filter SetHostname) Process(ctx *fasthttp.RequestCtx) {

	data, exists := filter.GetString("hostname")
	if exists {
		ctx.Request.SetHost(data)
	}
}

type SetResponse struct {
	param.Parameters
}

func (filter SetResponse) Name() string {
	return "set_response"
}

func (filter SetResponse) Process(ctx *fasthttp.RequestCtx) {

	status,hasStatus := filter.GetInt64("status",200)
	if hasStatus{
		ctx.Response.SetStatusCode(int(status))
	}

	contentType,hasContentType := filter.GetString("content_type")
	if hasContentType{
		ctx.SetContentType(contentType)
	}

	message,hasMessage := filter.GetString("body")
	if hasMessage{
		ctx.Response.SetBody([]byte(message))
	}
}
