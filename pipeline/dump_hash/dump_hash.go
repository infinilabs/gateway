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
	"github.com/OneOfOne/xxhash"
	"github.com/buger/jsonparser"
	xxhash1 "github.com/cespare/xxhash"
	log "github.com/cihub/seelog"
	xxhash2 "github.com/pierrec/xxHash/xxHash32"
	"github.com/segmentio/fasthash/fnv1a"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/progress"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"path"
	"sync"
)

type DumpHashProcessor struct {
	config Config
	client elastic.API
}

type Config struct {
	//字段名称必须是大写
	PartitionSize      int    `config:"partition_size"`
	BatchSize          int    `config:"batch_size"`
	SliceSize          int    `config:"slice_size"`
	Elasticsearch      string `config:"elasticsearch"`
	Output             string `config:"output_queue"`
	SortType           string `config:"sort_type"`
	SortField          string `config:"sort_field"`
	Indices            string `config:"indices"`
	Query              string `config:"query"`
	HashFunc           string `config:"hash_func"`
	ScrollTime         string `config:"scroll_time"`
	Fields             string `config:"fields"`
	SortDocumentFields bool   `config:"sort_document_fields"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		PartitionSize:  10,
		SliceSize:  1,
		BatchSize:  1000,
		ScrollTime: "5m",
		SortType:   "asc",
		HashFunc:   "xxhash32",
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

	//start := time.Now()
	wg := sync.WaitGroup{}

	file := path.Join(global.Env().GetDataDir(), "diff", processor.config.Output)
	if util.FileExists(file) {
		util.FileDelete(file)
		log.Warn("target file exists:", file, ",remove it now")
	}

	var totalDocsNeedToScroll int64 = 0
	for slice := 0; slice < processor.config.SliceSize; slice++ {

		tempSlice := slice

		ctx := pipeline.AcquireContext()
		wg.Add(1)
		go func(slice int, ctx *pipeline.Context) {
			defer wg.Done()

			scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, processor.config.Query, tempSlice, processor.config.SliceSize, processor.config.Fields, processor.config.SortField, processor.config.SortType)
			if err != nil {
				log.Errorf("%v-%v", processor.config.Output, err)
				return
			}

			initScrollID, err := jsonparser.GetString(scrollResponse1, "_scroll_id")
			if err != nil {
				log.Errorf("cannot get _scroll_id from json: %v, %v", string(scrollResponse1), err)
				return
			}

			version := processor.client.GetMajorVersion()
			totalHits, err := getScrollHitsTotal(version, scrollResponse1)
			if err != nil {
				panic(err)
			}

			totalDocsNeedToScroll += totalHits

			docSize := processor.processingDocs(scrollResponse1, processor.config.Output)

			progress.IncreaseWithTotal(processor.config.Output, "dump-hash-"+util.ToString(tempSlice), docSize, int(totalHits))

			log.Debugf("slice [%v] docs: %v / %v", tempSlice, docSize, totalHits)

			if totalHits == 0 {
				log.Tracef("slice %v is empty", tempSlice)
				return
			}

			var req = fasthttp.AcquireRequest()
			var res = fasthttp.AcquireResponse()
			defer fasthttp.ReleaseRequest(req)
			defer fasthttp.ReleaseResponse(res)
			var processedSize = 0
			for {

				if ctx.IsCanceled() {
					return
				}

				if initScrollID == "" {
					log.Errorf("[%v] scroll_id: [%v]", slice, initScrollID)
				}

				req.Reset()
				res.Reset()

				data, err := processor.client.NextScroll(req, res, processor.config.ScrollTime, initScrollID)

				if err != nil || len(data) == 0 {
					log.Error("failed to scroll,", processor.config.Elasticsearch, processor.config.Indices, string(data), err)
					return
				}

				if data != nil && len(data) > 0 {

					scrollID, err := jsonparser.GetString(data, "_scroll_id")
					if err != nil {
						log.Errorf("cannot get _scroll_id from json: %v, %v", string(scrollResponse1), err)
						return
					}

					var totalHits int64
					totalHits, err = getScrollHitsTotal(version, data)
					if err != nil {
						panic(err)
					}

					docSize := processor.processingDocs(data, processor.config.Output)

					processedSize += docSize
					log.Debugf("[%v] slice[%v]:%v,%v-%v", processor.config.Elasticsearch, slice, docSize, processedSize, totalHits)

					initScrollID = scrollID

					progress.IncreaseWithTotal(processor.config.Output, "dump-hash-"+util.ToString(tempSlice), docSize, int(totalHits))

					if docSize == 0 {
						log.Debugf("[%v] empty doc returned, break", slice)
						break
					}

				}

			}
			log.Debugf("%v - %v, slice %v is done", processor.config.Elasticsearch, processor.config.Indices, tempSlice)
		}(tempSlice, ctx)
	}

	progress.Start()
	wg.Wait()
	progress.Stop()

	//log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, stats, duration, math.Ceil(float64(stats)/math.Ceil((duration))))

	return nil
}

func getScrollHitsTotal(version int, data []byte) (int64, error) {
	if version >= 7 {
		return jsonparser.GetInt(data, "hits", "total", "value")
	} else {
		return jsonparser.GetInt(data, "hits", "total")
	}
}

func (processor *DumpHashProcessor) processingDocs(data []byte, outputQueueName string) int {

	buffer := bytebufferpool.Get()

	hashBuffer := bytebufferpool.Get()
	defer bytebufferpool.Put(hashBuffer)

	sourceBuffer := bytebufferpool.Get()
	defer bytebufferpool.Put(sourceBuffer)

	docSize := 0
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		id, err := jsonparser.GetString(value, "_id")
		if err != nil {
			panic(err)
		}

		source, _, _, err := jsonparser.Get(value, "_source")
		if err != nil {
			panic(err)
		}

		hash := processor.Hash(processor.config.HashFunc, hashBuffer, source)

		_, err = buffer.WriteBytesArray(util.UnsafeStringToBytes(id), []byte(","), hash, []byte("\n"))
		if err != nil {
			panic(err)
		}
		docSize++

	}, "hits", "hits")


	handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName), rotate.DefaultConfig)

	handler.Write(buffer.Bytes())

	bytebufferpool.Put(buffer)

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(docSize))

	return docSize
}

func (processor *DumpHashProcessor) Hash(hashFunc string, buf *bytebufferpool.ByteBuffer, data []byte) []byte {
	switch hashFunc {
	case "xxhash64":
		hash := xxhash1.Sum64(data)
		return []byte(util.Int64ToString(int64(hash)))
	case "xxhash32-1":
		hash := xxhash.New32()
		hash.Write(data)
		return []byte(util.Int64ToString(int64(hash.Sum32())))
	case "xxhash64-1":
		hash := xxhash.New64()
		hash.Write(data)
		return []byte(util.Int64ToString(int64(hash.Sum64())))
	case "xxhash32":
		h := xxhash2.New(0xCAFE)
		h.Write(data)
		r := h.Sum32()
		return []byte(util.Int64ToString(int64(r)))
	case "fnv1a":
		h1 := fnv1a.HashBytes32(data)
		return []byte(util.Int64ToString(int64(h1)))
	}

	h1 := fnv1a.HashBytes32(data)
	return []byte(util.Int64ToString(int64(h1)))
}
