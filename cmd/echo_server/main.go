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
		fmt.Println(string(ctx.Request.PhantomURI().Scheme()))
		fmt.Println(string(ctx.Request.PhantomURI().Host()))
		fmt.Println(string(ctx.Request.PhantomURI().FullURI()))
		fmt.Println(string(ctx.Request.PhantomURI().PathOriginal()))
		fmt.Println(string(ctx.Request.PhantomURI().QueryString()))
		fmt.Println(string(ctx.Request.PhantomURI().Hash()))
		fmt.Println(string(ctx.Request.PhantomURI().Username()))
		fmt.Println(string(ctx.Request.PhantomURI().Password()))
		fmt.Println(ctx.Request.Header.String(),true)
		fmt.Println(ctx.Request.GetRawBody(),true)
	}
	ctx.Response.Header.Set("SERVER",name)
	ctx.Response.SetStatusCode(200)
	fmt.Fprintf(ctx, ".")
}
