package common

import "infini.sh/framework/lib/fasthttp"

type Filter interface {
	Name() string
	Process(ctx *fasthttp.RequestCtx)
}

