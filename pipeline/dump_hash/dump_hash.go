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

package scroll

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/segmentio/fasthash/fnv1a"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"math"
	"path"
	"sync"
	"time"
)

type DumpHashProcessor struct {
	cfg    DumpHashConfig
	client elastic.API
}

type DumpHashConfig struct {
//字段名称必须是大写
	BatchSize int    `config:"batch_size"`
	SliceSize       int    `config:"slice_size"`
	Elasticsearch   string `config:"elasticsearch"`
	OutputQueueName string `config:"output_queue"`
	SortType  string `config:"sort_type"`
	SortField string `config:"sort_field"`
	Indices    string `config:"indices"`
	Query      string `config:"query"`
	ScrollTime string `config:"scroll_time"`
	Fields     string `config:"fields"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := DumpHashConfig{
		SliceSize:  1,
		BatchSize:  1000,
		ScrollTime: "5m",
		SortType:   "asc",
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of dump_hash processor: %s", err)
	}

	client := elastic.GetClient(cfg.Elasticsearch)
	if cfg.SliceSize < 1 || client.GetMajorVersion() < 5 {
		cfg.SliceSize = 1
	}

	return &DumpHashProcessor{
		cfg:    cfg,
		client: client,
	}, nil

}

func (joint *DumpHashProcessor) Name() string {
	return "dump_hash"
}

var scrollResponsePool = &sync.Pool{
	New: func() interface{} {
		c := elastic.ScrollResponse{}
		return &c
	},
}

var scrollResponseV7Pool = &sync.Pool{
	New: func() interface{} {
		c := elastic.ScrollResponseV7{}
		return &c
	},
}

func (joint *DumpHashProcessor) Process(c *pipeline.Context) error {

	start := time.Now()
	wg := sync.WaitGroup{}
	var statsLock sync.RWMutex
	var totalSize int

	file:=path.Join(global.Env().GetDataDir(), "diff", joint.cfg.OutputQueueName)
	if util.FileExists(file){
		log.Warn("target file exists:",file,",you may need to remove it first")
	}

	for slice := 0; slice < joint.cfg.SliceSize; slice++ {

		tempSlice := slice
		scrollResponse1, err := joint.client.NewScroll(joint.cfg.Indices, joint.cfg.ScrollTime, joint.cfg.BatchSize, joint.cfg.Query, tempSlice, joint.cfg.SliceSize, joint.cfg.Fields, joint.cfg.SortField, joint.cfg.SortType)
		if err != nil {
			log.Error(err)
			continue
		}

		docs := scrollResponse1.GetDocs()
		docSize := len(docs)

		statsLock.Lock()
		totalSize += docSize
		statsLock.Unlock()

		if docSize > 0 {
			processingDocs(docs, joint.cfg.OutputQueueName)
		}

		log.Debugf("slice %v docs: %v", tempSlice, scrollResponse1.GetHitsTotal())

		if scrollResponse1.GetHitsTotal() == 0 {
			log.Tracef("slice %v is empty", tempSlice)
			continue
		}
		wg.Add(1)

		go func() {
			defer wg.Done()
			var scrollResponse interface{}
			initScrollID := scrollResponse1.GetScrollId()

			version := joint.client.GetMajorVersion()

			for {
				data, err := joint.client.NextScroll(joint.cfg.ScrollTime, initScrollID)

				if err != nil {
					log.Error("failed to scroll,", joint.cfg.Elasticsearch, joint.cfg.Indices, string(data), err)
					return
				}

				if version >= 7 {
					scrollResponse = scrollResponseV7Pool.Get().(*elastic.ScrollResponseV7)

					err = scrollResponse.(*elastic.ScrollResponseV7).UnmarshalJSON(data)

					if err != nil {
						log.Error("failed to scroll,", joint.cfg.Elasticsearch, joint.cfg.Indices, string(data), err)
						return
					}
				} else {
					scrollResponse = scrollResponsePool.Get().(*elastic.ScrollResponse)
					err = scrollResponse.(*elastic.ScrollResponse).UnmarshalJSON(data)

					if err != nil {
						panic(err)
					}
				}

				obj, ok := scrollResponse.(elastic.ScrollResponseAPI)
				if !ok {
					if err != nil {
						panic(err)
					}
					break
				}

				initScrollID = obj.GetScrollId()

				docs := obj.GetDocs()
				docSize := len(docs)

				statsLock.Lock()
				totalSize += docSize
				stats.Gauge(fmt.Sprintf("scroll_total_received-%v", tempSlice), joint.cfg.OutputQueueName, int64(totalSize))
				statsLock.Unlock()

				if docSize == 0 {
					log.Trace(scrollResponse)
					break
				}

				processingDocs(docs, joint.cfg.OutputQueueName)

				if version >= 7 {
					scrollResponseV7Pool.Put(scrollResponse)
				} else {
					scrollResponsePool.Put(scrollResponse)
				}

			}
			log.Debugf("%v - %v, slice %v is done", joint.cfg.Elasticsearch, joint.cfg.Indices, tempSlice)

		}()

	}

	wg.Wait()

	duration := time.Now().Sub(start).Seconds()

	log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", joint.cfg.Elasticsearch, joint.cfg.Indices, totalSize, duration, math.Ceil(float64(totalSize)/math.Ceil((duration))))

	return nil
}

func processingDocs(docs []elastic.IndexDocument, outputQueueName string) {

	buffer := bytebufferpool.Get()

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(len(docs)))

	for _, v := range docs {

		h1 := fnv1a.HashBytes32(util.MustToJSONBytes(v.Source))

		_, err := buffer.WriteBytesArray([]byte((v.ID.(string))), []byte(","), []byte(util.Int64ToString(int64(h1))), []byte("\n"))
		if err != nil {
			panic(err)
		}
	}
	handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName), rotate.DefaultConfig)

	handler.Write(buffer.Bytes())
	bytebufferpool.Put(buffer)

}
