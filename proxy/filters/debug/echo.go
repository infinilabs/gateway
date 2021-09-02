/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package debug

import (
	"fmt"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type EchoMessage struct {
	param.Parameters
}

func (filter EchoMessage) Name() string {
	return "echo"
}

func (filter EchoMessage) Process(ctx *fasthttp.RequestCtx) {
	str := filter.GetStringOrDefault("str", ".")
	size := filter.GetIntOrDefault("repeat", 1)
	for i := 0; i < size; i++ {
		ctx.WriteString(str)
	}
	if !filter.GetBool("continue",true){
		ctx.Finished()
	}
}

type DumpHeader struct {
	param.Parameters
}

func (filter DumpHeader) Name() string {
	return "dump_header"
}

func (filter DumpHeader) Process(ctx *fasthttp.RequestCtx) {
	fmt.Println("request header:")
	fmt.Println(ctx.Request.Header.String())

	fmt.Println("response header:")
	fmt.Println("Status Code:",ctx.Response.StatusCode())
	fmt.Println(ctx.Response.Header.String())
}

type DumpUrl struct {
	param.Parameters
}

func (filter DumpUrl) Name() string {
	return "dump_url"
}

func (filter DumpUrl) Process(ctx *fasthttp.RequestCtx) {
	fmt.Println("request:\n ", ctx.Request.String())
	fmt.Println("\nuri: ", string(ctx.Request.RequestURI()))
	fmt.Println("uri: ", ctx.Request.URI().String())
	fmt.Println("query_args: ", ctx.Request.URI().QueryArgs().String())
	fmt.Println("query_string: ", string(ctx.Request.URI().QueryString()))
	_,user,pass:=ctx.Request.ParseBasicAuth()
	fmt.Println("username: ", string(user) )
	fmt.Println("password: ", string(pass) )
	_,apiID,apiKey:=ctx.ParseAPIKey()
	fmt.Println("api_id: ", string(apiID) )
	fmt.Println("api_key: ", string(apiKey) )
}



type DumpRequestBody struct {
	param.Parameters
}

func (filter DumpRequestBody) Name() string {
	return "dump_request_body"
}

func (filter DumpRequestBody) Process(ctx *fasthttp.RequestCtx) {
	fmt.Println("request_body: ", string(ctx.Request.Body()))
}


type DumpResponseBody struct {
	param.Parameters
}

func (filter DumpResponseBody) Name() string {
	return "dump_response_body"
}

func (filter DumpResponseBody) Process(ctx *fasthttp.RequestCtx) {
	fmt.Println("response_body: ", string(ctx.Response.GetRawBody()))
}

type DumpContext struct {
	param.Parameters
}

func (filter DumpContext) Name() string {
	return "dump_context"
}

func (filter DumpContext) Process(ctx *fasthttp.RequestCtx) {
	keys,ok:=filter.GetStringArray("context")
	if ok{
		fmt.Println("---- dumping context ---- ")
		for _,k:=range keys{
			v,err:=ctx.GetValue(k)
			if err!=nil{
				fmt.Println(k,", err:",err)
			}else{
				fmt.Println(k," : ", v)
			}
		}
	}
}
