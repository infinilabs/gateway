package common

import "infini.sh/framework/lib/fasthttp"

type RequestFilter interface {
	Filter
	Process(ctx *fasthttp.RequestCtx)
}

type Filter interface {
	Name() string
}


type ServiceFilter interface {
	Filter
	Process(ctx *fasthttp.RequestCtx)
}
