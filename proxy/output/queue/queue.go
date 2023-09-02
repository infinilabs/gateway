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
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"io"
	"strings"
	"sync"
)

type DiskEnqueueFilter struct {
	Type                         string                 `config:"type"`
	DepthThreshold               int64                  `config:"depth_threshold"`
	Message                      string                 `config:"message"` //override the message in the request
	QueueName                    string                 `config:"queue_name"`
	Labels                       map[string]interface{} `config:"labels,omitempty"`
	SaveMessageOffset            bool                   `config:"save_last_produced_message_offset,omitempty"`
	LastProducedMessageOffsetKey string                 `config:"last_produced_message_offset_key,omitempty"`
	messageBytes                 []byte
	queueNameTemplate            *fasttemplate.Template
	messageTemplate              *fasttemplate.Template
	labelsTemplate               map[string]*fasttemplate.Template
	producers                    sync.Map //map[string]queue.ProducerAPI
	qConfigs                     sync.Map // map[string]*queue.QueueConfig
}

func (filter *DiskEnqueueFilter) Name() string {
	return "queue"
}

func (filter *DiskEnqueueFilter) Filter(ctx *fasthttp.RequestCtx) {

	qName := filter.QueueName
	if filter.queueNameTemplate != nil {
		qName = filter.queueNameTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			variable, err := ctx.GetValue(tag)
			if err == nil {
				return w.Write([]byte(util.ToString(variable)))
			}
			return -1, err
		})
	}

	qConfig := filter.getQueueConfig(qName, ctx)

	if global.Env().IsDebug {
		log.Trace("get queue config:", qName, "->", qConfig)
	}

	if filter.DepthThreshold > 0 {
		depth := queue.Depth(qConfig)

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
		if filter.messageTemplate != nil {
			str := filter.messageTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
				variable, err := ctx.GetValue(tag)
				if err == nil {
					return w.Write([]byte(util.ToString(variable)))
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
	req := queue.ProduceRequest{Topic: qConfig.ID, Data: data}

	producer := filter.getProducer(qConfig)

	if producer == nil {
		panic(errors.New("invalid producer"))
	}

	res, err := producer.Produce(&[]queue.ProduceRequest{req})
	if err != nil || res == nil {
		panic(errors.Errorf("queue: %v, err: %v", qConfig, err))
	}

	offset := (*res)[0].Offset.String()

	if filter.SaveMessageOffset {
		ctx.PutValue(filter.LastProducedMessageOffsetKey, offset)
		ctx.Request.Header.Set(filter.LastProducedMessageOffsetKey, offset)
	}

}

func (filter *DiskEnqueueFilter) getProducer(qConfig *queue.QueueConfig) queue.ProducerAPI {
	if qConfig.ID == "" {
		panic(errors.Errorf("invalid queue config: %v", qConfig))
	}

	obj, ok := filter.producers.Load(qConfig.ID)
	if ok {
		return obj.(queue.ProducerAPI)
	}

	handler, err := queue.AcquireProducer(qConfig)
	if err != nil {
		panic(err)
	}
	filter.producers.Store(qConfig.ID, handler)
	return handler
}

func (filter *DiskEnqueueFilter) getQueueConfig(qName string, ctx *fasthttp.RequestCtx) *queue.QueueConfig {

	obj, ok := filter.qConfigs.Load(qName)
	if ok {
		qConfig, ok := obj.(*queue.QueueConfig)
		if ok {
			//log.Error("hit config cache:",qName, "->", qConfig)
			return qConfig
		}
	}

	labels := map[string]interface{}{}
	for k, v := range filter.Labels {
		labels[k] = v
	}

	if filter.labelsTemplate != nil && len(filter.labelsTemplate) > 0 {
		for k, v := range filter.labelsTemplate {
			label := v.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
				variable, err := ctx.GetValue(tag)
				if err == nil {
					return w.Write([]byte(util.ToString(variable)))
				}
				return -1, err
			})
			labels[k] = label
		}
	}

	tmp := queue.AdvancedGetOrInitConfig(filter.Type, qName, labels)
	if tmp != nil {
		queue.IniQueue(tmp)
	}

	if tmp == nil {
		panic(errors.Errorf("invalid queue config: %v", qName))
	}

	log.Trace("set config cache:", qName, "->", tmp)
	filter.qConfigs.Store(qName, tmp)
	return tmp
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("queue", NewDiskEnqueueFilter, &DiskEnqueueFilter{})
}

func NewDiskEnqueueFilter(c *config.Config) (pipeline.Filter, error) {

	runner := DiskEnqueueFilter{
		LastProducedMessageOffsetKey: "LAST_PRODUCED_MESSAGE_OFFSET",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if runner.Message != "" {
		runner.messageBytes = []byte(runner.Message)
		if strings.Contains(runner.Message, "$[[") {
			var err error
			runner.messageTemplate, err = fasttemplate.NewTemplate(runner.Message, "$[[", "]]")
			if err != nil {
				panic(err)
			}
		}
	}

	if strings.Contains(runner.QueueName, "$[[") {
		var err error
		runner.queueNameTemplate, err = fasttemplate.NewTemplate(runner.QueueName, "$[[", "]]")
		if err != nil {
			panic(err)
		}
	}

	runner.producers = sync.Map{} //map[string]queue.ProducerAPI{}
	runner.qConfigs = sync.Map{}  //map[string]*queue.QueueConfig{}
	runner.labelsTemplate = map[string]*fasttemplate.Template{}

	for k, v := range runner.Labels {
		str, ok := v.(string)
		if ok {
			if strings.Contains(str, "$[[") {
				var err error
				runner.labelsTemplate[k], err = fasttemplate.NewTemplate(str, "$[[", "]]")
				if err != nil {
					panic(err)
				}
			}
		}
	}

	return &runner, nil
}
