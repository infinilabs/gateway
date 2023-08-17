package queue

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/valyala/fasttemplate"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
	"io"
	"strings"
)

type DiskEnqueueFilter struct {
	Type           string                 `config:"type"`
	DepthThreshold int64                  `config:"depth_threshold"`
	Message        string                 `config:"message"` //override the message in the request
	QueueName      string                 `config:"queue_name"`
	Labels         map[string]interface{} `config:"labels,omitempty"`
	queueConfig    *queue.QueueConfig
	producer       queue.ProducerAPI
	messageBytes   []byte
	template       *fasttemplate.Template
}

func (filter *DiskEnqueueFilter) Name() string {
	return "queue"
}

func (filter *DiskEnqueueFilter) Filter(ctx *fasthttp.RequestCtx) {

	if filter.DepthThreshold > 0 {
		depth := queue.Depth(filter.queueConfig)

		if global.Env().IsDebug {
			log.Trace(filter.QueueName, " depth:", depth, " vs threshold:", filter.DepthThreshold)
		}

		if depth < filter.DepthThreshold {
			log.Warn("skip enqueue, ", filter.QueueName, " depth:", depth, " < threshold:", filter.DepthThreshold)
			return
		}
	}

	var data []byte
	if filter.messageBytes != nil {
		if filter.template != nil {
			str := filter.template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
				variable, err := ctx.GetValue(tag)
				x, ok := variable.(string)
				if ok {
					if x != "" {
						return w.Write([]byte(x))
					}
				}
				return -1, err
			})
			data = []byte(str)
		} else {
			data = filter.messageBytes
		}
	} else {
		data = ctx.Request.Encode()
	}

	var err error
	req := queue.ProduceRequest{Topic: filter.queueConfig.ID, Data: data}

	if filter.producer == nil {
		panic(errors.New("invalid producer"))
	}

	res, err := filter.producer.Produce(&[]queue.ProduceRequest{req})
	if err != nil || res == nil {
		panic(errors.Errorf("queue: %v, err: %v",filter.queueConfig,err))
	}

	offset := (*res)[0].Offset.String()

	ctx.PutValue("LAST_PRODUCED_MESSAGE_OFFSET", offset)
	ctx.Request.Header.Set("LAST_PRODUCED_MESSAGE_OFFSET", offset)

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("queue", NewDiskEnqueueFilter, &DiskEnqueueFilter{})
}

func NewDiskEnqueueFilter(c *config.Config) (pipeline.Filter, error) {

	runner := DiskEnqueueFilter{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.queueConfig = queue.GetOrInitConfig(runner.QueueName)

	if runner.queueConfig != nil {
		queue.IniQueue(runner.queueConfig)
	}

	if runner.Message != "" {
		runner.messageBytes = []byte(runner.Message)
		if strings.Contains(runner.Message, "$[[") {
			var err error
			runner.template, err = fasttemplate.NewTemplate(runner.Message, "$[[", "]]")
			if err != nil {
				panic(err)
			}
		}
	}

	handler, err := queue.AcquireProducer(runner.Type)
	if err != nil {
		panic(err)
	}
	runner.producer = handler

	return &runner, nil
}
