/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package elastic

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"net/http"
	log "github.com/cihub/seelog"
)

type BulkResponseValidate struct {
	param.Parameters
}

func (this BulkResponseValidate) Name() string {
	return "bulk_response_validate"
}

func (this BulkResponseValidate) Process(ctx *fasthttp.RequestCtx) {
	path:=string(ctx.URI().Path())
	if string(ctx.Request.Header.Method())!="POST"{
		return
	}

	if ctx.Response.StatusCode() == http.StatusOK && util.ContainStr(path,"_bulk") {
		data:=map[string]interface{}{}
		util.FromJSONBytes(ctx.Response.GetRawBody(),&data)
		err2,ok2:=data["errors"]
		if ok2{
			if err2==true{
				if global.Env().IsDebug{
					log.Error("invalid bulk response, ",ok2,",",err2,",",util.SubString(string(ctx.Response.GetRawBody()),0,256))
				}
				ctx.Response.SetStatusCode(this.GetIntOrDefault("invalid_status",500))
			}
		}
	}
}


