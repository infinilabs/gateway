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

package json_indexing

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/lib/bytebufferpool"
	"runtime"
	"sync"
	"time"
)

type IndexingMergeProcessor struct {
	bufferPool *bytebufferpool.Pool
	initLocker sync.RWMutex
	config     Config
}

//处理纯 json 格式的消息索引
func (processor *IndexingMergeProcessor) Name() string {
	return "json_indexing"
}

type Config struct {
	NumOfWorkers         int    `config:"worker_size"`
	IdleTimeoutInSeconds int    `config:"idle_timeout_in_seconds"`
	BulkSizeInKB         int    `config:"bulk_size_in_kb"`
	BulkSizeInMB         int    `config:"bulk_size_in_mb"`
	IndexName            string `config:"index_name"`
	TypeName             string `config:"type_name"`
	Elasticsearch        string `config:"elasticsearch"`
	InputQueue           string `config:"input_queue"`
	OutputQueue          string `config:"output_queue"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		BulkSizeInMB: 10,
		NumOfWorkers: 1,
		IdleTimeoutInSeconds: 5,
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of index_diff processor: %s", err)
	}

	diff := &IndexingMergeProcessor{
		config: cfg,
	}

	return diff, nil

}

//TODO 合并批量处理的操作，这里只用来合并请求和构造 bulk 请求。
//TODO 重启子进程，当子进程挂了之后
func (processor *IndexingMergeProcessor) Process(c *pipeline.Context) error {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error in json indexing,", v)
			}
		}
	}()

	bulkSizeInByte := 1048576 * processor.config.BulkSizeInMB
	if processor.config.BulkSizeInKB > 0 {
		bulkSizeInByte = 1024 * processor.config.BulkSizeInKB
	}

	if processor.bufferPool == nil {
		processor.initLocker.Lock()
		if processor.bufferPool == nil {
			estimatedBulkSizeInByte := bulkSizeInByte + (bulkSizeInByte / 3)
			processor.bufferPool = bytebufferpool.NewPool(uint64(estimatedBulkSizeInByte), uint64(bulkSizeInByte*2))
		}
		processor.initLocker.Unlock()
	}

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < processor.config.NumOfWorkers; i++ {
		wg.Add(1)
		go processor.NewBulkWorker(&totalSize, bulkSizeInByte, &wg)
	}

	wg.Wait()

	return nil
}

func (processor *IndexingMergeProcessor) NewBulkWorker(count *int, bulkSizeInByte int, wg *sync.WaitGroup) {

	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error in json indexing worker,", v)
				wg.Done()
			}
		}
	}()

	log.Trace("start bulk worker")

	mainBuf := processor.bufferPool.Get()
	mainBuf.Reset()
	defer processor.bufferPool.Put(mainBuf)
	docBuf := processor.bufferPool.Get()
	docBuf.Reset()
	defer processor.bufferPool.Put(docBuf)

	idleDuration := time.Duration(processor.config.IdleTimeoutInSeconds) * time.Second

	client := elastic.GetClient(processor.config.Elasticsearch)

	metadata:= elastic.GetMetadata(processor.config.Elasticsearch)

	var checkCount =0
	CHECK_AVAIABLE:
	if !metadata.IsAvailable(){
		checkCount++
		if checkCount>10{
			panic(errors.Errorf("cluster [%v] is not available",processor.config.Elasticsearch))
		}
		time.Sleep(10*time.Second)
		goto CHECK_AVAIABLE
	}

	if processor.config.TypeName == "" {
		if client.GetMajorVersion() < 7 {
			processor.config.TypeName = "doc"
		} else {
			processor.config.TypeName = "_doc"
		}
	}
	var lastCommit time.Time=time.Now()

READ_DOCS:
	for {
		pop, _, err := queue.PopTimeout(processor.config.InputQueue, idleDuration)
		if err != nil {
			log.Error(err)
			panic(err)
		}

		if len(pop)>0{
			stats.IncrementBy("bulk", "bytes_received", int64(mainBuf.Len()))

			docBuf.WriteString(fmt.Sprintf("{ \"index\" : { \"_index\" : \"%s\", \"_type\" : \"%s\" } }\n", processor.config.IndexName, processor.config.TypeName))
			docBuf.Write(pop)
			docBuf.WriteString("\n")

			mainBuf.Write(docBuf.Bytes())

			docBuf.Reset()
			(*count)++
		}

		//submit no matter the size of bulk after idle timeout
		if time.Since(lastCommit)>idleDuration && mainBuf.Len()>0{
			if global.Env().IsDebug {
				log.Trace("hit idle timeout, ", idleDuration.String())
			}
			goto CLEAN_BUFFER
		}

		if mainBuf.Len() > (bulkSizeInByte) {
			if global.Env().IsDebug {
				log.Trace("hit buffer size, ", mainBuf.Len())
			}
			goto CLEAN_BUFFER
		}

		goto READ_DOCS

	CLEAN_BUFFER:

		lastCommit=time.Now()

		if docBuf.Len() > 0 {
			mainBuf.Write(docBuf.Bytes())
		}

		if mainBuf.Len() > 0 {

			//TODO merge into bulk services
			mainBuf.WriteByte('\n')
			client.Bulk(mainBuf.Bytes())
			mainBuf.Reset()
			//TODO handle retry and fallback/over, dead letter queue
			//set services to failure, need manual restart
			//process dead letter queue first next round

			stats.IncrementBy("bulk", "bytes_processed", int64(mainBuf.Len()))
			log.Trace("clean buffer, and execute bulk insert")
		}

	}
}
