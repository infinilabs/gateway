// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package transform

import (
	"fmt"

	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type SetRequestHeader struct {
	Headers []string `config:"headers"`
	m       map[string]string
}

func (filter *SetRequestHeader) Name() string {
	return "set_request_header"
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_request_header", NewSetRequestHeader, &SetRequestHeader{})
}

func NewSetRequestHeader(c *config.Config) (pipeline.Filter, error) {

	runner := SetRequestHeader{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.m = map[string]string{}
	for _, item := range runner.Headers {
		k, v, err := util.ConvertStringToMap(item, "->")
		if err != nil {
			panic(err)
		}
		runner.m[k] = v
	}
	return &runner, nil
}

func (filter *SetRequestHeader) Filter(ctx *fasthttp.RequestCtx) {

	for k, v := range filter.m {
		//remove old one
		value := ctx.Request.Header.Peek(k)
		if len(value) > 0 {
			ctx.Request.Header.Del(k)
		}
		ctx.Request.Header.Set(k, v)
	}
}

type SetRequestQueryArgs struct {
	Args []string `config:"args"`
	m    map[string]string
}

func (filter *SetRequestQueryArgs) Name() string {
	return "set_request_query_args"
}

func (filter *SetRequestQueryArgs) Filter(ctx *fasthttp.RequestCtx) {
	clonedURI := ctx.Request.CloneURI()
	defer fasthttp.ReleaseURI(clonedURI)
	args := clonedURI.QueryArgs()
	for k, v := range filter.m {
		args.Set(k, v)
	}
	clonedURI.SetQueryString(args.String())
	ctx.Request.SetURI(clonedURI)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_request_query_args", NewSetRequestQueryArgs, &SetRequestQueryArgs{})
}

func NewSetRequestQueryArgs(c *config.Config) (pipeline.Filter, error) {

	runner := SetRequestQueryArgs{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.m = map[string]string{}
	for _, item := range runner.Args {
		k, v, err := util.ConvertStringToMap(item, "->")
		if err != nil {
			panic(err)
		}
		runner.m[k] = v
	}

	return &runner, nil
}

type SetResponseHeader struct {
	Headers []string `config:"headers"`
	m       map[string]string
}

func (filter *SetResponseHeader) Name() string {
	return "set_response_header"
}

func (filter *SetResponseHeader) Filter(ctx *fasthttp.RequestCtx) {

	for k, v := range filter.m {
		//remove old one
		value := ctx.Response.Header.Peek(k)
		if len(value) > 0 {
			ctx.Response.Header.Del(k)
		}
		ctx.Response.Header.Set(k, v)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_response_header", NewSetResponseHeader, &SetResponseHeader{})
}

func NewSetResponseHeader(c *config.Config) (pipeline.Filter, error) {

	runner := SetResponseHeader{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.m = map[string]string{}
	for _, item := range runner.Headers {
		k, v, err := util.ConvertStringToMap(item, "->")
		if err != nil {
			panic(err)
		}
		runner.m[k] = v
	}

	return &runner, nil
}

type SetHostname struct {
	Hostname string `config:"hostname"`
}

func (filter *SetHostname) Name() string {
	return "set_hostname"
}

func (filter *SetHostname) Filter(ctx *fasthttp.RequestCtx) {

	if filter.Hostname != "" {
		ctx.Request.SetHost(filter.Hostname)
		ctx.Request.Header.SetHost(filter.Hostname)
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_hostname", NewSetHostname, &SetHostname{})
}

func NewSetHostname(c *config.Config) (pipeline.Filter, error) {

	runner := SetHostname{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

type SetResponse struct {
	Status      int    `config:"status"`
	ContentType string `config:"content_type"`
	Body        string `config:"body"`
}

func (filter *SetResponse) Name() string {
	return "set_response"
}

func (filter *SetResponse) Filter(ctx *fasthttp.RequestCtx) {

	if filter.Status > 0 {
		ctx.Response.SetStatusCode(filter.Status)
	}

	if filter.ContentType != "" {
		ctx.SetContentType(filter.ContentType)
	}

	if filter.Body != "" {
		ctx.Response.SetBody(util.UnsafeStringToBytes(filter.Body))
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_response", NewSetResponse, &SetResponse{})
}

func NewSetResponse(c *config.Config) (pipeline.Filter, error) {

	runner := SetResponse{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
