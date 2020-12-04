package common

import "infini.sh/framework/lib/fasthttp"

type Filter interface {
	Name() string
}

type RequestFilter interface {
	Filter
	Process(ctx *fasthttp.RequestCtx)
}

type TCPFilter interface {
	Filter
}

type HeaderFilter interface {
	Filter
}

type ServiceFilter interface {
	Filter
	Process(ctx *fasthttp.RequestCtx)
}
