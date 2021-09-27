/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package elastic

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"net/http"
	"github.com/buger/jsonparser"
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
		va,err:=jsonparser.GetBoolean(ctx.Response.GetRawBody(),"errors")
		if va&&err==nil {
			if global.Env().IsDebug {
				log.Error("error in bulk requests,",util.SubString(string(ctx.Response.GetRawBody()), 0, 256))
			}
			ctx.Response.SetStatusCode(this.GetIntOrDefault("invalid_status",500))
		}
	}
}


