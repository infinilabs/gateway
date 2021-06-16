package auth

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type SetBasicAuth struct {
	param.Parameters
}

func (filter SetBasicAuth) Name() string {
	return "set_basic_auth"
}

func (filter SetBasicAuth) Process(ctx *fasthttp.RequestCtx) {
	username := filter.MustGetString("username")
	password := filter.MustGetString("password")

	//remove old one
	key,_:=ctx.Request.Header.PeekAnyKey(fasthttp.AuthHeaderKeys)
	if len(key)>0{
		ctx.Request.Header.Del(string(key))
	}

	//set new user
	ctx.Request.SetBasicAuth(username,password)
}
