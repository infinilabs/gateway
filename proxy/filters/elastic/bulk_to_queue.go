package elastic

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type BulkToQueue struct {
	param.Parameters
}

func (this BulkToQueue) Name() string {
	return "bulk_to_queue"
}

func (this BulkToQueue) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {
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

		ctx.Response.SetDestination(fmt.Sprintf("queue:%s",queueName))

		ctx.SetContentType(JSON_CONTENT_TYPE)
		ctx.WriteString("{\n  \"took\" : 0,\n  \"errors\" : false,\n  \"items\" : []\n}")
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
	}
}
