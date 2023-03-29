/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package dump

import (
	"fmt"

	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type DumpFilter struct {
	config *Config
}

type Config struct {
	Context []string `config:"context"`

	URI            bool `config:"uri"`
	Request        bool `config:"request"`
	Response       bool `config:"response"`
	QueryArgs      bool `config:"query_args"`
	User           bool `config:"user"`
	APIKey         bool `config:"api_key"`
	RequestHeader  bool `config:"request_header"`
	ResponseHeader bool `config:"response_header"`
	StatusCode     bool `config:"status_code"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("dump", New, &Config{})
}

func (filter *DumpFilter) Name() string {
	return "dump"
}

func (filter *DumpFilter) Filter(ctx *fasthttp.RequestCtx) {

	if filter.config.Request {
		fmt.Println("REQUEST:\n", util.TrimSpaces(ctx.Request.String()))
	}

	if filter.config.URI {
		fmt.Println("URI: ", ctx.Request.PhantomURI().String())
	}

	if filter.config.QueryArgs {
		fmt.Println("QUERY_ARGS: ", ctx.Request.PhantomURI().QueryArgs().String())
		fmt.Println("QUERY_STRING: ", string(ctx.Request.PhantomURI().QueryString()))
	}

	if filter.config.RequestHeader {
		fmt.Println("REQUEST_HEADER:")
		fmt.Println(ctx.Request.Header.String())
	}

	if filter.config.Response {
		fmt.Println("RESPONSE:\n", ctx.Response.String())
	}

	if filter.config.StatusCode {
		fmt.Println("STATUS_CODE:")
		fmt.Println(ctx.Response.StatusCode())
	}

	if filter.config.ResponseHeader {
		fmt.Println("RESPONSE_HEADER:")
		fmt.Println(ctx.Response.Header.String())
	}

	if filter.config.User {
		_, user, pass := ctx.Request.ParseBasicAuth()
		fmt.Println("USERNAME: ", string(user))
		fmt.Println("PASSWORD: ", string(pass))
	}

	if filter.config.APIKey {
		_, apiID, apiKey := ctx.ParseAPIKey()
		fmt.Println("API_ID: ", string(apiID))
		fmt.Println("API_KEY: ", string(apiKey))
	}

	if len(filter.config.Context) > 0 {
		fmt.Println("---- DUMPING CONTEXT ---- ")
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
