/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package http

import (
	"crypto/tls"
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	log "github.com/cihub/seelog"
)

type HTTPFilter struct {
	Schema string `config:"schema"`
	Host   string `config:"host"`
	client *fasthttp.Client
}

func (filter *HTTPFilter) Name() string {
	return "http"
}

func (filter *HTTPFilter) Filter(ctx *fasthttp.RequestCtx) {
	orignalHost:=string(ctx.Request.URI().Host())
	orignalSchema:=string(ctx.Request.URI().Scheme())
	ctx.Request.SetHost(filter.Host)
	ctx.Request.URI().SetScheme(filter.Schema)
	err:=filter.client.Do(&ctx.Request,&ctx.Response)
	if err!=nil{
		log.Error(err)
	}

	ctx.Request.URI().SetScheme(orignalSchema)
	ctx.Request.SetHost(orignalHost)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("http",NewHTTPFilter,&HTTPFilter{})
}

func NewHTTPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := HTTPFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.client=&fasthttp.Client{
	Name: "http_proxy",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &runner, nil
}
