package main

import (
	"flag"
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	"log"
	"runtime"
)

var port = flag.Int("port", 8080, "listening port")

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
	fmt.Fprintf(ctx, ".")
}
