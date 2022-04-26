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
	"time"
)

type HTTPFilter struct {
	requestTimeout time.Duration

	Timeout int `config:"timeout_in_second"`
	Schema string `config:"schema"`
	Host   string `config:"host"`
	Hosts   []string `config:"hosts"`
	client *fasthttp.Client
}

func (filter *HTTPFilter) Name() string {
	return "http"
}

func (filter *HTTPFilter) Filter(ctx *fasthttp.RequestCtx) {
	var err error
	for _,v:=range filter.Hosts{
		err=filter.forward(v,ctx)
		if err==nil{
			return
		}
	}
	if err!=nil{
		ctx.Response.SetBodyString(err.Error())
		return
	}
}

func (filter *HTTPFilter)forward(host string,ctx *fasthttp.RequestCtx)(err error){
	orignalHost:=string(ctx.Request.URI().Host())
	orignalSchema:=string(ctx.Request.URI().Scheme())
	ctx.Request.SetHost(host)
	ctx.Request.URI().SetScheme(filter.Schema)
	if filter.requestTimeout>0{
		err=filter.client.DoTimeout(&ctx.Request,&ctx.Response,filter.requestTimeout)
	}else{
		err=filter.client.Do(&ctx.Request,&ctx.Response)
	}
	ctx.Request.URI().SetScheme(orignalSchema)
	ctx.Request.SetHost(orignalHost)
	return err
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("http",NewHTTPFilter,&HTTPFilter{})
}

func NewHTTPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := HTTPFilter{
		Timeout: 10,
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.client=&fasthttp.Client{
		MaxConnsPerHost: 10000,
		Name: "http_proxy",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	runner.requestTimeout=time.Duration(runner.Timeout)*time.Second

	if runner.Host!=""{
		runner.Hosts=append(runner.Hosts,runner.Host)
	}

	return &runner, nil
}
