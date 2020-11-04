/*
Copyright 2016 Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package indexing

import (
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"math"
	"sync"
	"time"
)

type JsonBulkIndexingJoint struct {
	pipeline.Parameters
	inputQueueName string
}

func (joint JsonBulkIndexingJoint) Name() string {
	return "json_indexing"
}

func (joint JsonBulkIndexingJoint) Process(c *pipeline.Context) error {

	workers, _ := joint.GetInt("worker_size", 1)
	bulkSizeInMB, _ := joint.GetInt("bulk_size", 10)
	joint.inputQueueName = joint.GetStringOrDefault("input_queue", "es_queue")
	bulkSizeInMB = 1000000 * bulkSizeInMB

	start := time.Now()

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go joint.NewBulkWorker(&totalSize, bulkSizeInMB, &wg)
	}

	wg.Wait()

	duration := time.Now().Sub(start).Seconds()

	log.Info("bulk duration: ", duration, "s, ", "qps: ", math.Ceil(float64(totalSize)/math.Ceil((duration))))

	return nil
}

func (joint JsonBulkIndexingJoint) NewBulkWorker(count *int, bulkSizeInMB int, wg *sync.WaitGroup) {

	log.Trace("start bulk worker")

	mainBuf := bytes.Buffer{}
	docBuf := bytes.Buffer{}

	destIndex := joint.GetStringOrDefault("index_name", "")
	destType := joint.GetStringOrDefault("index_type", "_doc")
	//destType := joint.GetStringOrDefault("index_type", "")
	esInstanceVal := joint.GetStringOrDefault("elasticsearch", "es_json_bulk")

	client := elastic.GetClient(esInstanceVal)

READ_DOCS:
	for {
		select {
		case pop := <-queue.ReadChan(joint.inputQueueName):

			stats.IncrementBy("bulk", "event_received", int64(mainBuf.Len()))

			//TODO record ingest time,	request.LoggingTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

			docBuf.WriteString(fmt.Sprintf("{ \"index\" : { \"_index\" : \"%s\", \"_type\" : \"%s\" } }\n",destIndex,destType))
			docBuf.Write(pop)
			docBuf.WriteString("\n")

			mainBuf.Write(docBuf.Bytes())

			stats.Increment("pipeline", "bulk.hit")

			docBuf.Reset()
			(*count)++

			if mainBuf.Len() > (bulkSizeInMB) {
				log.Trace("hit buffer size, ", mainBuf.Len())
				goto CLEAN_BUFFER
			}

		case <-time.After(time.Second * 5):
			log.Trace("5s no message input")
			goto CLEAN_BUFFER
		}

		goto READ_DOCS

	CLEAN_BUFFER:

		if docBuf.Len() > 0 {
			mainBuf.Write(docBuf.Bytes())
		}

		if mainBuf.Len() > 0 {
			//fmt.Println(string(mainBuf.Bytes()))
			client.Bulk(&mainBuf)
			//TODO handle retry and fallback/over, dead letter queue
			//set services to failure, need manual restart
			//process dead letter queue first next round

			stats.IncrementBy("bulk", "event_processed", int64(mainBuf.Len()))
			log.Trace("clean buffer, and execute bulk insert")
		}

	}
}
