package common

import "infini.sh/framework/lib/fasthttp"

type Filter interface {
	Name() string
}

type RequestFilter interface {
	Filter
	Process(ctx *fasthttp.RequestCtx)
}
