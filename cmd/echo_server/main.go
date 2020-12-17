package main

import (
	"flag"
	"fmt"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	"log"
	"runtime"
)

var port = flag.Int("port", 8080, "listening port")
var debug = flag.Bool("debug", false, "dump request")
var name =util.PickRandomName()
func main() {
	runtime.GOMAXPROCS(1)
	flag.Parse()
	fmt.Printf("echo_server listen on: http://0.0.0.0:%v\n", *port)
	ln, err := reuseport.Listen("tcp4", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("error in reuseport listener: %s", err)
	}
	if err = fasthttp.Serve(ln, requestHandler); err != nil {
		log.Fatalf("error in fasthttp Server: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	if *debug{
		fmt.Println(string(ctx.Request.URI().Scheme()))
		fmt.Println(string(ctx.Request.URI().Host()))
		fmt.Println(string(ctx.Request.URI().FullURI()))
		fmt.Println(string(ctx.Request.URI().PathOriginal()))
		fmt.Println(string(ctx.Request.URI().QueryString()))
		fmt.Println(string(ctx.Request.URI().Hash()))
		fmt.Println(string(ctx.Request.URI().Username()))
		fmt.Println(string(ctx.Request.URI().Password()))
		fmt.Println(util.ToJson(ctx.Request.Header,true))
	}
	ctx.Response.Header.Set("SERVER",name)
	fmt.Fprintf(ctx, ".")
}
