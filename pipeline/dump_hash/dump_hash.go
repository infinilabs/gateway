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
	xxhash1 "github.com/cespare/xxhash"
	log "github.com/cihub/seelog"
	xxhash2 "github.com/pierrec/xxHash/xxHash32"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/valyala/fastjson"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/progress"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	fasthttp2 "infini.sh/framework/lib/fasthttp"
	"path"
	"github.com/OneOfOne/xxhash"
	"sync"
)

type DumpHashProcessor struct {
	config Config
	client elastic.API


}

type Config struct {
	//字段名称必须是大写
	BatchSize     int    `config:"batch_size"`
	SliceSize     int    `config:"slice_size"`
	Elasticsearch string `config:"elasticsearch"`
	Output        string `config:"output_queue"`
	SortType      string `config:"sort_type"`
	SortField     string `config:"sort_field"`
	Indices       string `config:"indices"`
	Query         string `config:"query"`
	HashFunc      string `config:"hash_func"`
	ScrollTime    string `config:"scroll_time"`
	Fields        string `config:"fields"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
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
	var pool = fastjson.ParserPool{}

	var totalDocsNeedToScroll = 0
	for slice := 0; slice < processor.config.SliceSize; slice++ {

		tempSlice := slice

		ctx:=pipeline.AcquireContext()
		wg.Add(1)
		go func(slice int,ctx *pipeline.Context) {
			defer wg.Done()

			scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, processor.config.Query, tempSlice, processor.config.SliceSize, processor.config.Fields, processor.config.SortField, processor.config.SortType)
			if err != nil {
				log.Errorf("%v-%v", processor.config.Output, err)
				return
			}
			var p = pool.Get()
			defer pool.Put(p)

			fastV,err := p.ParseBytes(scrollResponse1)
			if err != nil {
				log.Errorf("cannot parse json: %v, %v", string(scrollResponse1), err)
				return
			}
			initScrollID := util.UnsafeBytesToString(fastV.GetStringBytes("_scroll_id"))
			docs := fastV.GetArray("hits", "hits")

			version := processor.client.GetMajorVersion()
			totalHits := getScrollHitsTotal(version, fastV)

			totalDocsNeedToScroll+=totalHits

			docSize := len(docs)

			progress.IncreaseWithTotal(processor.config.Output,"dump-hash-"+util.ToString(tempSlice), docSize, totalHits)

			if docSize > 0 {
				processor.processingDocs(docs, processor.config.Output)
			}

			log.Debugf("slice [%v] docs: %v / %v", tempSlice, docSize,totalHits)

			if totalHits == 0 {
				log.Tracef("slice %v is empty", tempSlice)
				return
			}

			var req=fasthttp2.AcquireRequest()
			var res=fasthttp2.AcquireResponse()
			defer fasthttp2.ReleaseRequest(req)
			defer fasthttp2.ReleaseResponse(res)
			var processedSize = 0
			for {

				if ctx.IsCanceled(){
					return
				}

				if initScrollID == "" {
					log.Errorf("[%v] scroll_id: [%v]", slice, initScrollID)
				}

				req.Reset()
				res.Reset()

				data, err := processor.client.NextScroll(req,res,processor.config.ScrollTime, initScrollID)

				if err != nil || len(data) == 0 {
					log.Error("failed to scroll,", processor.config.Elasticsearch, processor.config.Indices, string(data), err)
					return
				}

				if data != nil && len(data) > 0 {
					fastV, err := p.ParseBytes(data)
					if err != nil {
						log.Errorf("cannot parse json: %v, %v", string(data), err)
						obj := map[string]interface{}{}
						util.FromJSONBytes(data, &obj)
						log.Error(obj["_scroll_id"])
						panic(err)
						return
					}
					scrollID := fastV.GetStringBytes("_scroll_id")
					hits := fastV.GetArray("hits", "hits")

					var totalHits int
					totalHits = getScrollHitsTotal(version, fastV)
					if version >= 7 {
						totalHits = fastV.GetInt("hits", "total", "value")
					} else {
						totalHits = fastV.GetInt("hits", "total")
					}

					processedSize += len(hits)
					log.Debugf("[%v] slice[%v]:%v,%v-%v", processor.config.Elasticsearch, slice, len(hits), processedSize, totalHits)

					initScrollID = util.UnsafeBytesToString(scrollID)
					docSize := len(hits)

					progress.IncreaseWithTotal(processor.config.Output,"dump-hash-"+util.ToString(tempSlice), docSize, totalHits)

					if docSize == 0 {
						log.Debugf("[%v] empty doc returned, break",slice)
						break
					}

					processor.processingDocs(hits, processor.config.Output)

				}

			}
			log.Debugf("%v - %v, slice %v is done", processor.config.Elasticsearch, processor.config.Indices, tempSlice)
		}(tempSlice,ctx)
	}

	progress.Start()
	wg.Wait()
	progress.Stop()

	//duration := time.Now().Sub(start).Seconds()

	//log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, stats, duration, math.Ceil(float64(stats)/math.Ceil((duration))))

	return nil
}

func getScrollHitsTotal(version int, fastV *fastjson.Value) int {
	if version >= 7 {
		return fastV.GetInt("hits", "total", "value")
	} else {
		return fastV.GetInt("hits", "total")
	}
}

func (processor *DumpHashProcessor) processingDocs(docs []*fastjson.Value, outputQueueName string) {

	buffer := bytebufferpool.Get()

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(len(docs)))

	hashBuffer := bytebufferpool.Get()
	defer bytebufferpool.Put(hashBuffer)

	for _, v := range docs {
		id := v.GetStringBytes("_id")

		source := v.GetObject("_source").String()

		hash := processor.Hash(processor.config.HashFunc, hashBuffer, util.UnsafeStringToBytes(source))
		//fmt.Println("hash:",string(hash),string(id))

		_, err := buffer.WriteBytesArray(id, []byte(","), hash, []byte("\n"))
		if err != nil {
			panic(err)
		}
	}

	handler := rotate.GetFileHandler(path.Join(global.Env().GetDataDir(), "diff", outputQueueName), rotate.DefaultConfig)
	handler.Write(buffer.Bytes())

	bytebufferpool.Put(buffer)

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
