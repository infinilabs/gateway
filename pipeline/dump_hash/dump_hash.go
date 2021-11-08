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
	"github.com/valyala/fastjson"
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
	"github.com/segmentio/fasthash/fnv1a"
	"sync"
	"time"
)

type DumpHashProcessor struct {
	config Config
	client elastic.API
}

type Config struct {
	//字段名称必须是大写
	BatchSize       int    `config:"batch_size"`
	SliceSize       int    `config:"slice_size"`
	Elasticsearch string `config:"elasticsearch"`
	Output        string `config:"output_queue"`
	SortType      string `config:"sort_type"`
	SortField       string `config:"sort_field"`
	Indices         string `config:"indices"`
	Query           string `config:"query"`
	ScrollTime      string `config:"scroll_time"`
	Fields          string `config:"fields"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
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
		config: cfg,
		client: client,
	}, nil

}

func (processor *DumpHashProcessor) Name() string {
	return "dump_hash"
}

func (processor *DumpHashProcessor) Process(c *pipeline.Context) error {

	start := time.Now()
	wg := sync.WaitGroup{}
	var statsLock sync.RWMutex
	var totalSize int

	file := path.Join(global.Env().GetDataDir(), "diff", processor.config.Output)
	if util.FileExists(file) {
		util.FileDelete(file)
		log.Warn("target file exists:", file, ",remove it now")
	}

	for slice := 0; slice < processor.config.SliceSize; slice++ {

		tempSlice := slice
		scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, processor.config.Query, tempSlice, processor.config.SliceSize, processor.config.Fields, processor.config.SortField, processor.config.SortType)
		if err != nil {
			log.Error(err)
			continue
		}

		fastV:= fastjson.MustParseBytes(scrollResponse1)
		initScrollID:=util.UnsafeBytesToString(fastV.GetStringBytes("_scroll_id"))
		docs:=fastV.GetArray("hits","hits")

		version := processor.client.GetMajorVersion()
		totalHits:=getScrollHitsTotal(version,fastV)

		docSize := len(docs)

		statsLock.Lock()
		totalSize += docSize
		statsLock.Unlock()

		if docSize > 0 {
			processor.processingDocs(docs, processor.config.Output)
		}

		log.Debugf("slice [%v] docs: %v / %v", tempSlice, totalSize,totalHits)

		if totalHits == 0 {
			log.Tracef("slice %v is empty", tempSlice)
			continue
		}
		wg.Add(1)

		go func(slice int) {
			defer wg.Done()
			var processedSize = 0
			for {
				data, err := processor.client.NextScroll(processor.config.ScrollTime, initScrollID)

				if err != nil ||len(data)==0 {
					log.Error("failed to scroll,", processor.config.Elasticsearch, processor.config.Indices, string(data), err)
					return
				}

				if data!=nil&&len(data)>0{

					fastV:= fastjson.MustParseBytes(data)
					scrollID:=fastV.GetStringBytes("_scroll_id")
					hits:=fastV.GetArray("hits","hits")

					var totalHits int
					totalHits=getScrollHitsTotal(version,fastV)
					if version >= 7 {
						totalHits= fastV.GetInt("hits","total","value")
					} else {
						totalHits= fastV.GetInt("hits","total")
					}

					processedSize+=len(hits)
					log.Debugf("[%v] slice[%v]:%v,%v-%v",processor.config.Elasticsearch,slice,len(hits),processedSize,totalHits)

					initScrollID = util.UnsafeBytesToString(scrollID)
					docSize := len(hits)
					statsLock.Lock()
					totalSize += docSize
					stats.Gauge(fmt.Sprintf("scroll_total_received-%v", tempSlice), processor.config.Output, int64(totalSize))
					statsLock.Unlock()

					if docSize == 0 {
						break
					}

					processor.processingDocs(hits, processor.config.Output)

				}

			}
			log.Debugf("%v - %v, slice %v is done", processor.config.Elasticsearch, processor.config.Indices, tempSlice)
		}(tempSlice)

	}

	wg.Wait()

	duration := time.Now().Sub(start).Seconds()

	log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, totalSize, duration, math.Ceil(float64(totalSize)/math.Ceil((duration))))

	return nil
}

func getScrollHitsTotal(version int,fastV *fastjson.Value)int {
	if version >= 7 {
		return fastV.GetInt("hits","total","value")
	} else {
		return fastV.GetInt("hits","total")
	}
}

func (processor *DumpHashProcessor) processingDocs(docs []*fastjson.Value, outputQueueName string) {

	buffer := bytebufferpool.Get()

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(len(docs)))

	for _, v := range docs {
		id:=v.GetStringBytes("_id")
		source:=v.GetObject("_source").String()
		h1 := fnv1a.HashBytes32(util.UnsafeStringToBytes(source))
		_, err := buffer.WriteBytesArray(id, []byte(","), []byte(util.Int64ToString(int64(h1))), []byte("\n"))
		if err != nil {
			panic(err)
		}
	}
	handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName), rotate.DefaultConfig)

	handler.Write(buffer.Bytes())
	bytebufferpool.Put(buffer)

}
