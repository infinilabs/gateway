package routing

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RedirectFilter struct {
	Uri string `config:"uri"`
}

func (filter *RedirectFilter) Name() string {
	return "redirect"
}

func (filter *RedirectFilter) Filter(ctx *fasthttp.RequestCtx) {
	ctx.Redirect(filter.Uri,302)
	ctx.Finished()
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("redirect",NewRedirectFilter,&RedirectFilter{})
}

func NewRedirectFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RedirectFilter{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
