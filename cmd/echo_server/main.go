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
