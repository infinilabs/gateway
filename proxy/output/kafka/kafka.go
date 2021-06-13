package kafka

import (
	"context"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"src/github.com/segmentio/kafka-go"
	"sync"
	"time"
)

type Kafka struct {
	param.Parameters
}

func (filter Kafka) Name() string {
	return "to_kafka"
}


var msgPool *sync.Pool

func initPool() {
	if msgPool!=nil{
		return
	}
	msgPool = &sync.Pool {
		New: func()interface{} {
			return kafka.Message{}
		},
	}
}

var inited bool
var taskContext = context.Background()
var w *kafka.Writer
var messages=[]kafka.Message{}
var batchSize int
var lock sync.Mutex

func (filter Kafka) Process(ctx *fasthttp.RequestCtx) {

	topic:=filter.GetStringOrDefault("topic","infini-gateway")
	brokers:=filter.MustGetStringArray("brokers")
	if !inited {
		initPool()
		batchSize=filter.GetIntOrDefault("batch_size",1000)
		batchTimeout:=filter.GetIntOrDefault("batch_timeout_in_ms",500)
		requiredAcks:=filter.GetIntOrDefault("required_acks",0)
		w = kafka.NewWriter(kafka.WriterConfig{
			Brokers: brokers,
			Topic:   topic,
			BatchSize: batchSize,
			BatchTimeout: time.Duration(batchTimeout) * time.Millisecond,
			RequiredAcks: requiredAcks,
		})
		inited = true
	}

	msg:=msgPool.Get().(kafka.Message)

	msg.Key=ctx.Request.RequestURI()
	msg.Value=ctx.Request.Body()

	lock.Lock()

	messages=append(messages,msg)

	//check need to flush or not
	if len(messages)>=batchSize{
		err := w.WriteMessages(taskContext, messages...)
		if err != nil {
			panic("could not write message " + err.Error())
		}
		for _,v:=range messages{
			msgPool.Put(v)
		}
		messages=[]kafka.Message{}
	}
	lock.Unlock()
}
