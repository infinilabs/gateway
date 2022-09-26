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

package es_scroll

import (
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/progress"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"math"
	"runtime"
	"sync"
	"time"
)

type ScrollProcessor struct {
	config Config
	client elastic.API
}

type Config struct {
	//字段名称必须是大写
	PartitionSize      int    `config:"partition_size"`
	BatchSize          int    `config:"batch_size"`
	SliceSize          int    `config:"slice_size"`
	Elasticsearch      string `config:"elasticsearch"`
	SortType           string `config:"sort_type"`
	SortField          string `config:"sort_field"`
	Indices            string `config:"indices"`
	QueryString              string `config:"query_string"`
	QueryDSL              string `config:"query_dsl"`
	ScrollTime         string `config:"scroll_time"`
	Fields             string `config:"fields"`
	Output             string `config:"output_queue"`

	RemoveTypeMeta         bool         `config:"remove_type"`
	//RemovePipeline         bool         `config:"remove_pipeline"`
	IndexNameRename  map[string]string `config:"index_rename"`
	TypeNameRename   map[string]string `config:"type_rename"`
}

func init() {
	pipeline.RegisterProcessorPlugin("es_scroll", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		PartitionSize:  10,
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

	return &ScrollProcessor{
		config: cfg,
		client: client,
	}, nil

}

func (processor *ScrollProcessor) Name() string {
	return "es_scroll"
}

func (processor *ScrollProcessor) Process(c *pipeline.Context) error {

	start := time.Now()
	wg := sync.WaitGroup{}

	var totalDocsNeedToScroll int64 = 0
	for slice := 0; slice < processor.config.SliceSize; slice++ {

		tempSlice := slice
		progress.RegisterBar(processor.config.Output, "scroll-"+util.ToString(tempSlice), 100)

		wg.Add(1)
		go func(slice int, ctx *pipeline.Context) {
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
						log.Error("error in processor,", v)
					}
				}
				log.Debug("exit detector for active queue")
				wg.Done()
			}()

			scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, processor.config.QueryString, tempSlice, processor.config.SliceSize, processor.config.Fields, processor.config.SortField, processor.config.SortType)
			if err != nil {
				log.Errorf("%v-%v", processor.config.Output, err)
				panic(err)
			}

			initScrollID, err := jsonparser.GetString(scrollResponse1, "_scroll_id")
			if err != nil {
				log.Errorf("cannot get _scroll_id from json: %v, %v", string(scrollResponse1), err)
				panic(err)
			}

			version := processor.client.GetMajorVersion()
			totalHits, err := common.GetScrollHitsTotal(version, scrollResponse1)
			if err != nil {
				panic(err)
			}

			totalDocsNeedToScroll += totalHits

			docSize := processor.processingDocs(scrollResponse1, processor.config.Output)

			progress.IncreaseWithTotal(processor.config.Output, "scroll-"+util.ToString(tempSlice), docSize, int(totalHits))

			log.Debugf("slice [%v] docs: %v / %v", tempSlice, docSize, totalHits)

			if totalHits == 0 {
				log.Tracef("slice %v is empty", tempSlice)
				return
			}

			var req = fasthttp.AcquireRequest()
			var res = fasthttp.AcquireResponse()
			defer fasthttp.ReleaseRequest(req)
			defer fasthttp.ReleaseResponse(res)

			meta:=elastic.GetMetadata(processor.config.Elasticsearch)
			apiCtx:=&elastic.APIContext{
				Client: meta.GetHttpClient(meta.GetActivePreferredHost("")),
				Request: req,
				Response: res,
			}

			var processedSize = 0
			for {

				if ctx.IsCanceled() {
					return
				}

				if initScrollID == "" {
					log.Errorf("[%v] scroll_id: [%v]", slice, initScrollID)
				}

				apiCtx.Request.Reset()
				apiCtx.Response.Reset()

				data, err := processor.client.NextScroll(apiCtx, processor.config.ScrollTime, initScrollID)

				if err != nil || len(data) == 0 {
					log.Error("failed to scroll,slice:",slice,",", processor.config.Elasticsearch,",", processor.config.Indices,",", string(data),",", err)
					panic(err)
				}

				if data != nil && len(data) > 0 {

					scrollID, err := jsonparser.GetString(data, "_scroll_id")
					if err != nil {
						log.Errorf("cannot get _scroll_id from json: %v, %v", string(scrollResponse1), err)
						panic(err)
					}

					var totalHits int64
					totalHits, err = common.GetScrollHitsTotal(version, data)
					if err != nil {
						panic(err)
					}

					docSize := processor.processingDocs(data, processor.config.Output)

					processedSize += docSize
					log.Debugf("[%v] slice[%v]:%v,%v-%v", processor.config.Elasticsearch, slice, docSize, processedSize, totalHits)

					initScrollID = scrollID

					progress.IncreaseWithTotal(processor.config.Output, "scroll-"+util.ToString(tempSlice), docSize, int(totalHits))

					if docSize == 0 {
						log.Debugf("[%v] empty doc returned, break", slice)
						break
					}

				}

			}
			log.Debugf("%v - %v, slice %v is done", processor.config.Elasticsearch, processor.config.Indices, tempSlice)
		}(tempSlice, c)
	}

	progress.Start()
	wg.Wait()
	progress.Stop()

	duration:=time.Since(start).Seconds()
	log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, totalDocsNeedToScroll, duration, math.Ceil(float64(totalDocsNeedToScroll)/math.Ceil((duration))))

	return nil
}

func (processor *ScrollProcessor) processingDocs(data []byte, outputQueueName string) int {

	hashBuffer := bytebufferpool.Get("es_scroll")
	defer bytebufferpool.Put("es_scroll",hashBuffer)

	docSize := 0
	var docs=map[int]*bytebufferpool.ByteBuffer{}
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		id, err := jsonparser.GetString(value, "_id")
		if err != nil {
			panic(err)
		}

		index, err := jsonparser.GetString(value, "_index")
		if err != nil {
			panic(err)
		}

		typeStr, err := jsonparser.GetString(value, "_type")
		if err != nil {
			panic(err)
		}

		if index != "" && len(processor.config.IndexNameRename) > 0 {
			v, ok := processor.config.IndexNameRename[index]
			if ok {
				index = v
			} else {
				v, ok := processor.config.IndexNameRename["*"]
				if ok {
					index = v
				}
			}
		}

		if typeStr != "" &&!processor.config.RemoveTypeMeta && len(processor.config.TypeNameRename) > 0 {
			v, ok := processor.config.TypeNameRename[typeStr]
			if ok&& v!=typeStr {
				typeStr = v
			} else {
				v, ok := processor.config.TypeNameRename["*"]
				if ok && v!=typeStr {
					typeStr = v
				}
			}
		}

		source, _, _, err := jsonparser.Get(value, "_source")
		if err != nil {
			panic(err)
		}
		stats.Increment("scrolling_docs","docs")
		//stats.IncrementBy("scrolling_docs","size", int64(len(source)))

		//hash := processor.Hash(processor.config.HashFunc, hashBuffer, source)

		partitionID:=elastic.GetShardID(7,util.UnsafeStringToBytes(id),processor.config.PartitionSize)

		buffer,ok:=docs[partitionID]
		if !ok{
			buffer = bytebufferpool.Get("es_scroll")
		}

		//trim newline to space
		util.WalkBytesAndReplace(source,util.NEWLINE,util.SPACE)

		buffer.WriteString(fmt.Sprintf("{ \"index\" : { \"_index\" : \"%s\", \"_type\" : \"%s\", \"_id\" : \"%s\" } }\n", index, typeStr,id))
		buffer.Write(source)
		buffer.WriteString("\n")

		//_, err = buffer.WriteBytesArray(util.UnsafeStringToBytes(id), []byte(","), hash, []byte("\n"))
		//if err != nil {
		//	panic(err)
		//}

		docSize++

		docs[partitionID]=buffer

	}, "hits", "hits")

	for k,v:=range docs{
		queueConfig := &queue.QueueConfig{}
		queueConfig.Source = "dynamic"
		queueConfig.Labels = util.MapStr{}
		queueConfig.Labels["type"] = "scroll_docs"
		queueConfig.Name=outputQueueName+util.ToString(k)
		queue.RegisterConfig(queueConfig.Name, queueConfig)

		queue.Push(queue.GetOrInitConfig(outputQueueName+util.ToString(k)),v.Bytes())

		//handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName+util.ToString(k)), rotate.DefaultConfig)
		//handler.Write(v.Bytes())
		bytebufferpool.Put("es_scroll",v)
	}

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(docSize))

	return docSize
}
