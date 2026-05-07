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
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	es_common "infini.sh/framework/modules/elastic/common"
	"infini.sh/gateway/common"
)

type ScrollProcessor struct {
	config         Config
	client         elastic.API
	clientID       string
	requestTimeout time.Duration
	HTTPPool       *fasthttp.RequestResponsePool
}

type OutputQueueConfig struct {
	Name   string                 `config:"name"`
	Labels map[string]interface{} `config:"labels"`
}

type Config struct {
	Elasticsearch       string                       `config:"elasticsearch"`
	ElasticsearchConfig *elastic.ElasticsearchConfig `config:"elasticsearch_config"`

	//字段名称必须是大写
	PartitionSize int    `config:"partition_size"`
	BatchSize     int    `config:"batch_size"`
	SliceSize     int    `config:"slice_size"`
	SortType      string `config:"sort_type"`
	SortField     string `config:"sort_field"`
	BulkOperation string `config:"bulk_operation"`
	Indices       string `config:"indices"`
	QueryString   string `config:"query_string"`
	QueryDSL      string `config:"query_dsl"`
	ScrollTime    string `config:"scroll_time"`
	Fields        string `config:"fields"`
	// DEPRECATED, use `queue` instead
	Output string             `config:"output_queue"`
	Queue  *OutputQueueConfig `config:"queue"`

	RemoveTypeMeta bool `config:"remove_type"`
	//RemovePipeline         bool         `config:"remove_pipeline"`
	IndexNameRename map[string]string `config:"index_rename"`
	TypeNameRename  map[string]string `config:"type_rename"`
}

func truncateLogValue(value string, limit int) string {
	if value == "" || limit <= 0 || len(value) <= limit {
		return value
	}
	return util.SubString(value, 0, limit) + "..."
}

func (processor *ScrollProcessor) getRequestTimeout() time.Duration {
	if processor.requestTimeout > 0 {
		return processor.requestTimeout
	}

	meta := elastic.GetMetadata(processor.clientID)
	if meta == nil || meta.Config == nil || meta.Config.RequestTimeout <= 0 {
		return 0
	}
	return time.Duration(meta.Config.RequestTimeout) * time.Second
}

func (processor *ScrollProcessor) buildScrollErrorContext(slice int, scrollID string, apiCtx *elastic.APIContext) string {
	parts := []string{
		fmt.Sprintf("cluster=%s", processor.config.Elasticsearch),
		fmt.Sprintf("indices=%s", processor.config.Indices),
		fmt.Sprintf("slice=%d/%d", slice, processor.config.SliceSize),
		fmt.Sprintf("scroll=%s", processor.config.ScrollTime),
		fmt.Sprintf("batch_size=%d", processor.config.BatchSize),
	}

	if timeout := processor.getRequestTimeout(); timeout > 0 {
		parts = append(parts, fmt.Sprintf("request_timeout=%s", timeout))
	}

	if scrollID != "" {
		parts = append(parts, fmt.Sprintf("scroll_id_prefix=%s", truncateLogValue(scrollID, 64)))
	}

	if apiCtx != nil && apiCtx.Request != nil {
		if method := util.UnsafeBytesToString(apiCtx.Request.Header.Method()); method != "" {
			parts = append(parts, fmt.Sprintf("method=%s", method))
		}
		if host := util.UnsafeBytesToString(apiCtx.Request.Host()); host != "" {
			parts = append(parts, fmt.Sprintf("host=%s", host))
		}
		if requestURI := util.UnsafeBytesToString(apiCtx.Request.RequestURI()); requestURI != "" {
			parts = append(parts, fmt.Sprintf("request_uri=%s", truncateLogValue(requestURI, 256)))
		}
		if apiCtx.Response != nil {
			if statusCode := apiCtx.Response.StatusCode(); statusCode > 0 {
				parts = append(parts, fmt.Sprintf("status=%d", statusCode))
			}
		}
	}

	return strings.Join(parts, ", ")
}

func (processor *ScrollProcessor) wrapScrollRequestError(action string, slice int, err error, query *elastic.SearchRequest, response []byte, scrollID string, apiCtx *elastic.APIContext) error {
	if err == nil {
		err = fmt.Errorf("empty scroll response")
	}

	contextParts := []string{processor.buildScrollErrorContext(slice, scrollID, apiCtx)}

	if processor.config.QueryString != "" {
		contextParts = append(contextParts, fmt.Sprintf("query_string=%s", truncateLogValue(processor.config.QueryString, 512)))
	}
	if processor.config.QueryDSL != "" {
		contextParts = append(contextParts, fmt.Sprintf("query_dsl=%s", truncateLogValue(processor.config.QueryDSL, 1024)))
	}
	if query != nil {
		contextParts = append(contextParts, fmt.Sprintf("search_request=%s", truncateLogValue(util.MustToJSON(query), 1024)))
	}
	if len(response) > 0 {
		contextParts = append(contextParts, fmt.Sprintf("response=%s", truncateLogValue(util.UnsafeBytesToString(response), 1024)))
	} else {
		contextParts = append(contextParts, "response=<empty>")
	}

	return fmt.Errorf("%s failed (%s): %w", action, strings.Join(contextParts, ", "), err)
}

func (processor *ScrollProcessor) recordScrollRequestStart(ctx *pipeline.Context, stage string) int64 {
	startTime := time.Now().UnixMilli()
	ctx.PutValue("es_scroll.last_request_stage", stage)
	ctx.PutValue("es_scroll.last_request_start_time", startTime)
	return startTime
}

func (processor *ScrollProcessor) recordScrollRequestDone(ctx *pipeline.Context, startedAt int64) {
	if startedAt <= 0 {
		return
	}
	ctx.PutValue("es_scroll.last_request_duration_ms", time.Now().UnixMilli()-startedAt)
}

func (processor *ScrollProcessor) recordSuccessfulExport(ctx *pipeline.Context) {
	ctx.PutValue("es_scroll.last_successful_export_time", time.Now().UnixMilli())
}

func init() {
	pipeline.RegisterProcessorPlugin("es_scroll", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		PartitionSize: 1,
		SliceSize:     1,
		BatchSize:     1000,
		ScrollTime:    "5m",
		SortType:      "asc",
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of dump_hash processor: %s", err)
	}

	if cfg.Queue != nil {
		cfg.Output = cfg.Queue.Name
	}

	client := elastic.GetClientNoPanic(cfg.Elasticsearch)
	if client == nil {
		if cfg.ElasticsearchConfig != nil {
			cfg.ElasticsearchConfig.Source = "es_scroll"
			client, _ = es_common.InitElasticInstanceWithoutMetadata(*cfg.ElasticsearchConfig)
		}
	}
	if client == nil {
		panic("failed to get elasticsearch client")
	}

	esVersion := client.GetVersion()

	if cfg.SliceSize < 1 {
		cfg.SliceSize = 1
	}
	if esVersion.Distribution == elastic.Elasticsearch && esVersion.Major < 5 {
		cfg.SliceSize = 1
	}

	requestTimeout := time.Duration(0)
	if cfg.ElasticsearchConfig != nil && cfg.ElasticsearchConfig.RequestTimeout > 0 {
		requestTimeout = time.Duration(cfg.ElasticsearchConfig.RequestTimeout) * time.Second
	}

	return &ScrollProcessor{
		config:         cfg,
		client:         client,
		clientID:       cfg.Elasticsearch,
		requestTimeout: requestTimeout,
		HTTPPool:       fasthttp.NewRequestResponsePool("es_scroll"),
	}, nil

}

func (processor *ScrollProcessor) Name() string {
	return "es_scroll"
}

func (processor *ScrollProcessor) Process(c *pipeline.Context) error {

	start := time.Now()
	wg := sync.WaitGroup{}

	var totalDocsNeedToScroll int64 = 0
	var totalDocsScrolled int64 = 0

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
						ctx.RecordError(fmt.Errorf("es_scroll panic: %v", r))
					}
				}
				log.Debug("exit detector for active queue")
				wg.Done()
			}()

			var query *elastic.SearchRequest = elastic.GetSearchRequest(processor.config.QueryString, processor.config.QueryDSL, processor.config.Fields, processor.config.SortField, processor.config.SortType)
			query = common.EnsureExactScrollTotalHits(processor.client.GetVersion(), query)
			requestStartTime := processor.recordScrollRequestStart(ctx, "new_scroll")
			scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, query, tempSlice, processor.config.SliceSize)
			processor.recordScrollRequestDone(ctx, requestStartTime)
			if err != nil {
				wrappedErr := processor.wrapScrollRequestError("new scroll", tempSlice, err, query, nil, "", nil)
				log.Error(wrappedErr)
				panic(wrappedErr)
			}

			initScrollID, err := jsonparser.GetString(scrollResponse1, "_scroll_id")
			if err != nil {
				log.Errorf("cannot get _scroll_id from json: %v, %v", util.SubString(string(scrollResponse1), 0, 1024), err)
				panic(err)
			}

			defer func() {
				err := processor.client.ClearScroll(initScrollID)
				if err != nil {
					log.Errorf("failed to clear scroll, err: %v", err)
				}
			}()

			version := processor.client.GetVersion()
			totalHits, err := common.GetScrollHitsTotal(version, scrollResponse1)
			if err != nil {
				log.Errorf("cannot get total hits from json: %v, %v", util.SubString(string(scrollResponse1), 0, 1024), err)
				panic(err)
			}

			atomic.AddInt64(&totalDocsNeedToScroll, totalHits)
			ctx.PutValue("es_scroll.total_hits", atomic.LoadInt64(&totalDocsNeedToScroll))

			docSize := processor.processingDocs(scrollResponse1, processor.config.Output)
			atomic.AddInt64(&totalDocsScrolled, int64(docSize))
			ctx.PutValue("es_scroll.scrolled_docs", atomic.LoadInt64(&totalDocsScrolled))
			if docSize > 0 {
				processor.recordSuccessfulExport(ctx)
			}

			progress.IncreaseWithTotal(processor.config.Output, "scroll-"+util.ToString(tempSlice), docSize, int(totalHits))

			log.Debugf("slice [%v] docs: %v / %v", tempSlice, docSize, totalHits)

			if totalHits == 0 {
				log.Tracef("slice %v is empty", tempSlice)
				return
			}

			var req = processor.HTTPPool.AcquireRequest()
			var res = processor.HTTPPool.AcquireResponse()
			defer processor.HTTPPool.ReleaseRequest(req)
			defer processor.HTTPPool.ReleaseResponse(res)

			meta := elastic.GetMetadata(processor.config.Elasticsearch)
			apiCtx := &elastic.APIContext{
				Client:   meta.GetHttpClient(meta.GetActivePreferredHost("")),
				Request:  req,
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

				requestStartTime := processor.recordScrollRequestStart(ctx, "next_scroll")
				data, err := processor.client.NextScroll(apiCtx, processor.config.ScrollTime, initScrollID)
				processor.recordScrollRequestDone(ctx, requestStartTime)

				if err != nil || len(data) == 0 {
					wrappedErr := processor.wrapScrollRequestError("next scroll", slice, err, nil, data, initScrollID, apiCtx)
					log.Error(wrappedErr)
					panic(wrappedErr)
				}

				if data != nil && len(data) > 0 {

					scrollID, err := jsonparser.GetString(data, "_scroll_id")
					if err != nil {
						log.Errorf("cannot get _scroll_id from json: %v, %v", util.SubString(string(scrollResponse1), 0, 1024), err)
						panic(err)
					}

					var totalHits int64
					totalHits, err = common.GetScrollHitsTotal(version, data)
					if err != nil {
						panic(err)
					}

					docSize := processor.processingDocs(data, processor.config.Output)
					atomic.AddInt64(&totalDocsScrolled, int64(docSize))
					ctx.PutValue("es_scroll.scrolled_docs", atomic.LoadInt64(&totalDocsScrolled))
					if docSize > 0 {
						processor.recordSuccessfulExport(ctx)
					}

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

	duration := time.Since(start).Seconds()
	log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, totalDocsNeedToScroll, duration, math.Ceil(float64(totalDocsNeedToScroll)/math.Ceil((duration))))

	return nil
}

func (processor *ScrollProcessor) processingDocs(data []byte, outputQueueName string) int {

	docSize := 0
	var docs = map[int]*bytebufferpool.ByteBuffer{}
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
			log.Debugf("get _type field error: %v", err)
		}
		routing, err := jsonparser.GetString(value, "_routing")
		if err != nil {
			log.Debugf("get _routing field error: %v", err)
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

		if processor.config.RemoveTypeMeta {
			// delete _type field from batch output
			typeStr = ""
		} else if len(processor.config.TypeNameRename) > 0 {
			// Support rename any (including empty) source _type with *: doc mapping
			v, ok := processor.config.TypeNameRename[typeStr]
			if ok && v != typeStr {
				typeStr = v
			} else {
				v, ok := processor.config.TypeNameRename["*"]
				if ok && v != typeStr {
					typeStr = v
				}
			}
		}

		source, _, _, err := jsonparser.Get(value, "_source")
		if err != nil {
			panic(err)
		}
		stats.Increment("scrolling_docs", "docs")
		//stats.IncrementBy("scrolling_docs","size", int64(len(source)))

		//hash := processor.Hash(processor.config.HashFunc, hashBuffer, source)

		partitionID := elastic.GetShardID(7, util.UnsafeStringToBytes(id), processor.config.PartitionSize)

		buffer, ok := docs[partitionID]
		if !ok {
			buffer = bytebufferpool.Get("es_scroll")
		}

		//trim newline to space
		util.WalkBytesAndReplace(source, util.NEWLINE, util.SPACE)

		bulkOperation := "index"
		if len(processor.config.BulkOperation) > 0 {
			bulkOperation = processor.config.BulkOperation
		}
		buffer.WriteString(fmt.Sprintf("{ \"%s\" : { \"_index\" : \"%s\", ", bulkOperation, index))
		if typeStr != "" {
			buffer.WriteString(fmt.Sprintf("\"_type\" : \"%s\",", typeStr))
		}
		if routing != "" {
			buffer.WriteString(fmt.Sprintf("\"routing\" : \"%s\",", routing))
		}
		buffer.WriteString(fmt.Sprintf("\"_id\" : \"%s\" } }\n", id))
		buffer.Write(source)
		buffer.WriteString("\n")

		//_, err = buffer.WriteBytesArray(util.UnsafeStringToBytes(id), []byte(","), hash, []byte("\n"))
		//if err != nil {
		//	panic(err)
		//}

		docSize++

		docs[partitionID] = buffer

	}, "hits", "hits")

	for k, v := range docs {
		queueConfig := &queue.QueueConfig{}
		queueConfig.Source = "dynamic"
		queueConfig.Labels = util.MapStr{}
		queueConfig.Labels["type"] = "scroll_docs"
		if processor.config.Queue != nil {
			for k, v := range processor.config.Queue.Labels {
				queueConfig.Labels[k] = v
			}
		}
		queueConfig.Name = outputQueueName + util.ToString(k)
		queue.RegisterConfig(queueConfig)
		pushQueue := queue.GetOrInitConfig(outputQueueName + util.ToString(k))
		if global.Env().IsDebug {
			log.Trace("queue config: ", pushQueue)
		}

		if err := queue.Push(pushQueue, v.Bytes()); err != nil {
			log.Errorf("failed to push data to queue: %v, %v", outputQueueName+util.ToString(k), err)
			panic(err)
		}

		//handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName+util.ToString(k)), rotate.DefaultConfig)
		//handler.Write(v.Bytes())
		bytebufferpool.Put("es_scroll", v)
	}

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(docSize))

	return docSize
}
