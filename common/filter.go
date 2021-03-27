package common

import (
	"infini.sh/framework/lib/fasthttp"
)

type RequestFilter interface {
	Name() string
	Process(filterCfg *FilterConfig,ctx *fasthttp.RequestCtx)
}
