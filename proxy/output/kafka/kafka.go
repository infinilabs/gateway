// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
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

	msgPool     *sync.Pool
	taskContext context.Context
	messages    []kafka.Message
	lock        sync.Mutex
	w           *kafka.Writer
}

func (filter *Kafka) Name() string {
	return "kafka"
}

func (filter *Kafka) Filter(ctx *fasthttp.RequestCtx) {

	msg := filter.msgPool.Get().(kafka.Message)
	msg.Key = util.Int64ToBytes(int64(util.GetIncrementID64("request")))
	msg.Value = ctx.Request.Encode()

	filter.lock.Lock()
	filter.messages = append(filter.messages, msg)

	//check need to flush or not
	if len(filter.messages) >= filter.BatchSize {
		err := filter.w.WriteMessages(filter.taskContext, filter.messages...)
		if err != nil {
			panic("could not write message " + err.Error())
		}
		for _, v := range filter.messages {
			filter.msgPool.Put(v)
		}
		filter.messages = []kafka.Message{}
	}

	filter.lock.Unlock()
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("kafka", NewKafkaFilter, &Kafka{})
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

	runner.w = kafka.NewWriter(kafka.WriterConfig{
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

	runner.taskContext = context.Background()
	runner.messages = []kafka.Message{}
	runner.lock = sync.Mutex{}

	return &runner, nil
}
