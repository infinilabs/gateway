/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

type RequestBodyJsonDel struct {
	param.Parameters
}

func (filter RequestBodyJsonDel) Name() string {
	return "request_body_json_del"
}

func (filter RequestBodyJsonDel) Process(ctx *fasthttp.RequestCtx) {
	bodyBytes:=ctx.Request.GetRawBody()
	ignoreNotFound:=filter.GetBool("ignore_missing",false)
	paths, exists := filter.GetStringArray("path")
	if exists {
		if len(bodyBytes)==0{
			bodyBytes=[]byte("{}")
		}

		for _,path:=range paths{
			pathArray:=strings.Split(path,".")
			v,t,offset,err:=jsonparser.Get(bodyBytes,pathArray...)
			if t==jsonparser.NotExist&&ignoreNotFound{
				log.Debugf("path:%v, %v, %v, %v, %v",path,err,v,t,offset)
				continue
			}

			bodyBytes=jsonparser.Delete(bodyBytes,pathArray...)
			if err!=nil{
				log.Errorf("path:%v, %v",path,err)
				return
			}
		}
		ctx.Request.SetBody(bodyBytes)
		return
	}

}

