package disk_queue

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
)

type DiskEnqueueFilter struct {
	param.Parameters
}

func (filter DiskEnqueueFilter) Name() string {
	return "disk_queue"
}

func (filter DiskEnqueueFilter) Process(ctx *fasthttp.RequestCtx) {
	threshold := filter.GetInt64OrDefault("depth_threshold",0)
	queueName := filter.MustGetString("queue_name")
	depth:=queue.Depth(queueName)

	if global.Env().IsDebug{
		log.Trace(queueName," depth:",depth," vs threshold:",threshold)
	}

	if depth>=threshold{
		data:=ctx.Request.Encode()
		err:=queue.Push(queueName,data)
		if err!=nil{
			panic(err)
		}
	}else{
		log.Debug("skip enqueue, ",queueName," depth:",depth," vs threshold:",threshold)
	}
}

