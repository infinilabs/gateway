package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"sync"
	"time"
)

type Kafka struct {
	Topic            string   `config:"topic"`
	BatchSize        int      `config:"batch_size"`
	BatchTimeoutInMs int      `config:"batch_timeout_in_ms"`
	RequiredAcks     int      `config:"required_acks"`
	Brokers          []string `config:"brokers"`

	msgPool *sync.Pool
}

func (filter *Kafka) Name() string {
	return "kafka"
}

var taskContext = context.Background()
var w *kafka.Writer
var messages = []kafka.Message{}
var lock sync.Mutex

func (filter *Kafka) Filter(ctx *fasthttp.RequestCtx) {

	msg := filter.msgPool.Get().(kafka.Message)
	msg.Key = ctx.Request.RequestURI()
	msg.Value = ctx.Request.Body()

	lock.Lock()

	messages = append(messages, msg)

	//TODO flush finally or periodly
	//check need to flush or not
	if len(messages) >= filter.BatchSize {
		err := w.WriteMessages(taskContext, messages...)
		if err != nil {
			panic("could not write message " + err.Error())
		}
		for _, v := range messages {
			filter.msgPool.Put(v)
		}
		messages = []kafka.Message{}
	}

	lock.Unlock()
}

func NewKafkaFilter(c *config.Config) (pipeline.Filter, error) {

	runner := Kafka{
		BatchSize:        1000,
		BatchTimeoutInMs: 500,
		RequiredAcks:     0,
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	w = kafka.NewWriter(kafka.WriterConfig{
		Brokers:      runner.Brokers,
		Topic:        runner.Topic,
		BatchSize:    runner.BatchSize,
		BatchTimeout: time.Duration(runner.BatchTimeoutInMs) * time.Millisecond,
		RequiredAcks: runner.RequiredAcks,
	})

	runner.msgPool = &sync.Pool{
		New: func() interface{} {
			return kafka.Message{}
		},
	}

	return &runner, nil
}
