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

//https://www.sohamkamani.com/golang/working-with-kafka/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"github.com/segmentio/kafka-go"
	"strconv"
	"time"
)

const (
	topic         = "cbp5cm3q50k10squ2na0"
	brokerAddress = "192.168.3.188:9092"
)

func main() {
	// create a new context
	ctx := context.Background()
	// produce messages in a new go routine, since
	// both the produce and consume functions are
	// blocking

	for i:=0;i<10;i++{
		go produce(ctx)
	}
	//consume(ctx)


	time.Sleep(1*time.Hour)
}

func produce(ctx context.Context) {
	// initialize a counter
	i := 0

	// intialize the writer with the broker addresses, and the topic
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddress},
		Topic:   topic,
		BatchSize: 1000,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: 0,
		// assign the logger to the writer
	})

	w.AllowAutoTopicCreation=true

	messages:=[]kafka.Message{}
	j:=0
	for {

		for j=0;j<1000;j++{
			msg:=kafka.Message{
				Key: []byte(strconv.Itoa(i)),
				Value: []byte("this is message" + strconv.Itoa(i)),
			}
			messages=append(messages,msg)
		}

		err := w.WriteMessages(ctx, messages...)
		if err != nil {
			panic("could not write message " + err.Error())
		}
		//fmt.Print(".")
		messages=[]kafka.Message{}
		//fmt.Println("writes:", i)
		i++
	}
}

func consume(ctx context.Context) {
	// create a new logger that outputs to stdout
	// and has the `kafka reader` prefix
	l := log.New(os.Stdout, "kafka reader: ", 0)
	// initialize a new reader with the brokers and topic
	// the groupID identifies the consumer and prevents
	// it from receiving duplicate messages
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddress},
		Topic:   topic,
		GroupID: "my-group",
		// assign the logger to the reader
		Logger: l,
	})
	for {
		// the `ReadMessage` method blocks until we receive the next event
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			panic("could not read message " + err.Error())
		}
		// after receiving the message, log its value
		fmt.Println("received: ", string(msg.Value))
	}
}
