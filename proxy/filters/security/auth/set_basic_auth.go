package auth

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type SetBasicAuth struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

func (filter *SetBasicAuth) Name() string {
	return "set_basic_auth"
}

func (filter *SetBasicAuth) Filter(ctx *fasthttp.RequestCtx) {

	//remove old one
	key, _ := ctx.Request.Header.PeekAnyKey(fasthttp.AuthHeaderKeys)
	if len(key) > 0 {
		ctx.Request.Header.Del(string(key))
	}

	//set new user
	ctx.Request.SetBasicAuth(filter.Username, filter.Password)
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_basic_auth",NewSetBasicAuth,&SetBasicAuth{})
}

func NewSetBasicAuth(c *config.Config) (pipeline.Filter, error) {

	runner := SetBasicAuth{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
