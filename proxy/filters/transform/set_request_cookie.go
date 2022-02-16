/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package transform

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type Cookie struct {
	Reset bool `config:"reset"`//reset request cookies
	Cookies map[string]string `config:"cookies"`//request cookies
}

func (filter *Cookie) Name() string {
	return "set_request_cookie"
}

func (filter *Cookie) Filter(ctx *fasthttp.RequestCtx) {
	if filter.Reset{
		ctx.Request.Header.DelAllCookies()
	}

	for k,v:=range filter.Cookies{
		ctx.Request.Header.SetCookie(k,v)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_request_cookie", NewCookieFilter,&Cookie{})
}

func NewCookieFilter(c *config.Config) (pipeline.Filter, error) {

	runner := Cookie{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
