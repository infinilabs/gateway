/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package dump

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type DumpFilter struct {
	config *Config
}

type Config struct {
	Context []string `config:"context"`

	URI            bool `config:"uri"`
	Request        bool `config:"request"`
	QueryArgs      bool `config:"query_args"`
	User           bool `config:"user"`
	APIKey         bool `config:"api_key"`
	RequestHeader  bool `config:"request_header"`
	ResponseHeader bool `config:"response_header"`
	StatusCode bool `config:"status_code"`
}

func (filter *DumpFilter) Name() string {
	return "dump"
}

func (filter *DumpFilter) Filter(ctx *fasthttp.RequestCtx) {

	if filter.config.Request {
		fmt.Println("request:\n ", ctx.Request.String())
	}

	if filter.config.URI {
		fmt.Println("uri: ", ctx.Request.URI().String())
	}

	if filter.config.QueryArgs {
		fmt.Println("query_args: ", ctx.Request.URI().QueryArgs().String())
		fmt.Println("query_string: ", string(ctx.Request.URI().QueryString()))
	}

	if filter.config.RequestHeader {
		fmt.Println("request header:")
		fmt.Println(ctx.Request.Header.String())
	}

	if filter.config.StatusCode {
		fmt.Println("response status code:")
		fmt.Println(ctx.Response.StatusCode())
	}

	if filter.config.ResponseHeader {
		fmt.Println("response header:")
		fmt.Println(ctx.Response.Header.String())
	}

	if filter.config.User {
		_, user, pass := ctx.Request.ParseBasicAuth()
		fmt.Println("username: ", string(user))
		fmt.Println("password: ", string(pass))
	}

	if filter.config.APIKey {
		_, apiID, apiKey := ctx.ParseAPIKey()
		fmt.Println("api_id: ", string(apiID))
		fmt.Println("api_key: ", string(apiKey))
	}

	if len(filter.config.Context) > 0 {
		fmt.Println("---- dumping context ---- ")
		for _, k := range filter.config.Context {
			v, err := ctx.GetValue(k)
			if err != nil {
				fmt.Println(k, ", err:", err)
			} else {
				fmt.Println(k, " : ", v)
			}
		}
	}

}

func New(c *config.Config) (pipeline.Filter, error) {

	cfg := Config{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := DumpFilter{config: &cfg}

	return &runner, nil
}
