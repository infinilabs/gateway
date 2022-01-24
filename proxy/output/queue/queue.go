package queue

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
)

type DiskEnqueueFilter struct {
	DepthThreshold int64  `config:"depth_threshold"`
	QueueName      string `config:"queue_name"`
}

func (filter *DiskEnqueueFilter) Name() string {
	return "queue"
}

func (filter *DiskEnqueueFilter) Filter(ctx *fasthttp.RequestCtx) {
	qCfg := queue.GetOrInitConfig(filter.QueueName)

	depth := queue.Depth(qCfg)

	if global.Env().IsDebug {
		log.Trace(filter.QueueName, " depth:", depth, " vs threshold:", filter.DepthThreshold)
	}

	if depth >= filter.DepthThreshold {
		data := ctx.Request.Encode()
		err := queue.Push(qCfg, data)
		if err != nil {
			panic(err)
		}
	} else {
		log.Debug("skip enqueue, ", filter.QueueName, " depth:", depth, " vs threshold:", filter.DepthThreshold)
	}
}

func init() {
	pipeline.RegisterFilterPlugin("queue",NewDiskEnqueueFilter)
}

func NewDiskEnqueueFilter(c *config.Config) (pipeline.Filter, error) {

	runner := DiskEnqueueFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
