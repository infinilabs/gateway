package elastic

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
)

type BulkToQueue struct {
	param.Parameters
}

func (this BulkToQueue) Name() string {
	return "bulk_to_queue"
}

func (this BulkToQueue) Process(ctx *fasthttp.RequestCtx) {
	ctx.Set(common.CACHEABLE, false)
	clusterName:=this.MustGetString("elasticsearch")
	path:=string(ctx.URI().Path())

	if util.PrefixStr(path,"/_bulk"){
		body:=ctx.Request.GetRawBody()
		queueName:=fmt.Sprintf("%v_bulk",clusterName)
		err:=queue.Push(queueName,body)
		if err!=nil{
			log.Error(err)
			return
		}

		ctx.SetDestination(fmt.Sprintf("queue:%s",queueName))

		ctx.SetContentType(JSON_CONTENT_TYPE)
		ctx.WriteString("{\n  \"took\" : 0,\n  \"errors\" : false,\n  \"items\" : []\n}")
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
	}
}


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
					log.Error("checking bulk response, invalid, ",ok2,",",err2,",",util.SubString(string(ctx.Response.GetRawBody()),0,256))
				}
				ctx.Response.SetStatusCode(this.GetIntOrDefault("invalid_status",500))
			}
		}
	}
}


