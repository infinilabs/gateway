package main

import (
	"flag"
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	"log"
	"runtime"
)

//func fastHTTPHandler(ctx *fasthttp.RequestCtx) {
//	fmt.Fprintf(ctx, "%q", ctx.RequestURI())
//}
//func main() {
//
//	runtime.GOMAXPROCS(runtime.NumCPU())
//
//	//// pass bound struct method to fasthttp
//	//myHandler := &MyHandler{
//	//	foobar: "foobar",
//	//}
//	//fasthttp.ListenAndServe(":8080", myHandler.HandleFastHTTP)
//
//	// pass plain function to fasthttp
//	//s := &fasthttp.ListenAndServe(":8081", fastHTTPHandler)
//	s := &fasthttp.Server{
//		FallbackHandler:     fastHTTPHandler,
//		//ReadTimeout:          time.Hour,
//		//WriteTimeout:         time.Hour,
//		//MaxKeepaliveDuration:0,
//		//IdleTimeout:0,
//		//Concurrency: fasthttp.DefaultConcurrency,
//	}
//	//s.DisableKeepalive = true // If this is false, we see the error randomly.
//
//	go func() {
//		s.ListenAndServe(":8081")
//	}()
//	go func() {
//		s.ListenAndServe(":8082")
//	}()
//	go func() {
//		s.ListenAndServe(":8083")
//	}()
//
//	log.Fatal(s.ListenAndServe(":8084"))
//
//}

var port = flag.Int("port", 8080, "listening port")

func main() {
	runtime.GOMAXPROCS(1)
	flag.Parse()
	fmt.Println("listen on: 0.0.0.0:", *port)
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

//➜  es wrk   -c 100 -d 30s -t 1 -H --latency  http://localhost:8082/_cat/health
//Running 30s test @ http://localhost:8082/_cat/health
//1 threads and 100 connections
//Thread Stats   Avg      Stdev     Max   +/- Stdev
//Latency   502.18us  100.81us   1.79ms   65.66%
//Req/Sec   160.23k     8.73k  170.19k    79.07%
//4798362 requests in 30.10s, 617.77MB read
//Requests/sec: 159406.06
//Transfer/sec:     20.52MB

//hc := fasthttp.Client{
//MaxConnsPerHost: 60000,
//TLSConfig:       &tls.Config{InsecureSkipVerify: true},
//}
//req := fasthttp.AcquireRequest()
//req.SetBody([]byte(`{"username":"xxxxxx", "password":"xxxxxx"}`))
//req.Header.SetContentType("application/x-www-form-urlencoded")
//req.Header.SetMethod("POST")
//resp := fasthttp.AcquireResponse()
//req.SetRequestURI("https://127.0.0.1:8060/fip/v1/auth/login")
//defer fasthttp.ReleaseRequest(req)
//defer fasthttp.ReleaseResponse(resp)
//if err := hc.DoTimeout(req, resp, 20*time.Second); err != nil {
//}
//
//fmt.Println(string(resp.Body()))
