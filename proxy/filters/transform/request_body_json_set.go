/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"fmt"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"strings"
)

type RequestBodyJsonSet struct {
	param.Parameters
}

func (filter RequestBodyJsonSet) Name() string {
	return "request_body_json_set"
}

func (filter RequestBodyJsonSet) Process(ctx *fasthttp.RequestCtx) {

	bodyBytes:=ctx.Request.GetRawBody()

	ignoreNotFound:=filter.GetBool("ignore_missing",false)
	paths, exists := filter.GetStringMap("path")
	//var err error
	if exists {
		if len(bodyBytes)==0{
			bodyBytes=[]byte("{}")
		}

		for path,value:=range paths{
			pathArray:=strings.Split(path,".")
			v,t,offset,err:=jsonparser.Get(bodyBytes,pathArray...)
			if t==jsonparser.NotExist&&ignoreNotFound{
				log.Debugf("path:%v, value:%v, %v, %v, %v, %v",path,value,err,v,t,offset)
				continue
			}

			bodyBytes,err=jsonparser.Set(bodyBytes,[]byte(value),pathArray...)
			if err!=nil{
				log.Errorf("path:%v, value:%v, %v",path,value,err)
				return
			}
		}
		ctx.Request.SetBody(bodyBytes)
		return
	}

	obj,existJson:=filter.GetMap("json")
	if existJson{
		fmt.Println(obj)
	}

}

