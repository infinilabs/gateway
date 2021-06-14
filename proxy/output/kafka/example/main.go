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
	topic         = "medcl"
	brokerAddress = "localhost:9092"
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
