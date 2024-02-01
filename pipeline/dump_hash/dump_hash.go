/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dump_hash

import (
	"fmt"
	"math"
	"path"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

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
	es_common "infini.sh/framework/modules/elastic/common"
	"infini.sh/gateway/common"
)

type DumpHashProcessor struct {
	config   Config
	files    sync.Map
	client   elastic.API
	HTTPPool *fasthttp.RequestResponsePool
}

type Config struct {
	//字段名称必须是大写
	PartitionSize       int                          `config:"partition_size"`
	BatchSize           int                          `config:"batch_size"`
	SliceSize           int                          `config:"slice_size"`
	Elasticsearch       string                       `config:"elasticsearch"`
	ElasticsearchConfig *elastic.ElasticsearchConfig `config:"elasticsearch_config"`
	Output              string                       `config:"output_queue"`
	SortType            string                       `config:"sort_type"`
	SortField           string                       `config:"sort_field"`
	Indices             string                       `config:"indices"`
	CleanOldFiles       bool                         `config:"clean_old_files"`
	KeepSourceInResult  bool                         `config:"keep_source"`

	QueryString string `config:"query_string"`
	QueryDSL    string `config:"query_dsl"`

	HashFunc     string              `config:"hash_func"`
	ScrollTime   string              `config:"scroll_time"`
	Fields       string              `config:"fields"`
	RotateConfig rotate.RotateConfig `config:"rotate"`

	//SortDocumentFields bool   `config:"sort_document_fields"`
}

func init() {
	pipeline.RegisterProcessorPlugin("dump_hash", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		PartitionSize: 10,
		SliceSize:     1,
		BatchSize:     1000,
		ScrollTime:    "5m",
		SortType:      "asc",
		HashFunc:      "xxhash32",
		RotateConfig: rotate.RotateConfig{
			Compress:     false,
			MaxFileAge:   0,
			MaxFileCount: 0,
			MaxFileSize:  1024 * 1000,
		},
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of dump_hash processor: %s", err)
	}

	client := elastic.GetClientNoPanic(cfg.Elasticsearch)
	if client == nil {
		if cfg.ElasticsearchConfig != nil {
			cfg.ElasticsearchConfig.Source = "dump_hash"
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

	return &DumpHashProcessor{
		config:   cfg,
		client:   client,
		files:    sync.Map{},
		HTTPPool: fasthttp.NewRequestResponsePool("dump_hash"),
	}, nil

}

func (processor *DumpHashProcessor) Name() string {
	return "dump_hash"
}

func (processor *DumpHashProcessor) Process(c *pipeline.Context) error {

	start := time.Now()
	wg := sync.WaitGroup{}

	if processor.config.CleanOldFiles {
		for i := 0; i < processor.config.PartitionSize; i += 1 {
			file := path.Join(global.Env().GetDataDir(), "diff", processor.config.Output+"-"+strconv.Itoa(i))
			err := util.FileDelete(file)
			log.Infof("deleting old dump file [%s], err: %v", file, err)
		}
	}

	var totalDocsNeedToScroll int64 = 0
	var totalDocsScrolled int64 = 0

	for slice := 0; slice < processor.config.SliceSize; slice++ {

		tempSlice := slice
		progress.RegisterBar(processor.config.Output, "dump-hash-"+util.ToString(tempSlice), 100)

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

			var query *elastic.SearchRequest = elastic.GetSearchRequest(processor.config.QueryString, processor.config.QueryDSL, processor.config.Fields, processor.config.SortField, processor.config.SortType)
			scrollResponse1, err := processor.client.NewScroll(processor.config.Indices, processor.config.ScrollTime, processor.config.BatchSize, query, tempSlice, processor.config.SliceSize)
			if err != nil {
				log.Errorf("%v-%v", processor.config.Output, err)
				panic(err)
			}

			initScrollID, err := jsonparser.GetString(scrollResponse1, "_scroll_id")
			if err != nil {
				log.Errorf("cannot get _scroll_id from json: %v, %v", util.SubString(string(scrollResponse1), 0, 1024), err)
				panic(err)
			}

			version := processor.client.GetVersion()
			totalHits, err := common.GetScrollHitsTotal(version, scrollResponse1)
			if err != nil {
				panic(err)
			}

			atomic.AddInt64(&totalDocsNeedToScroll, totalHits)
			ctx.PutValue("dump_hash.total_hits", atomic.LoadInt64(&totalDocsNeedToScroll))

			docSize := processor.processingDocs(scrollResponse1, processor.config.Output)
			atomic.AddInt64(&totalDocsScrolled, int64(docSize))
			ctx.PutValue("dump_hash.scrolled_docs", atomic.LoadInt64(&totalDocsScrolled))

			progress.IncreaseWithTotal(processor.config.Output, "dump-hash-"+util.ToString(tempSlice), docSize, int(totalHits))

			log.Debugf("slice [%v] docs: %v / %v", tempSlice, docSize, totalHits)

			if totalHits == 0 || docSize == 0 {

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

				data, err := processor.client.NextScroll(apiCtx, processor.config.ScrollTime, initScrollID)

				if err != nil || len(data) == 0 {
					log.Error("failed to scroll,", processor.config.Elasticsearch, processor.config.Indices, string(data), err)
					panic(err)
				}

				if len(data) > 0 {

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
					ctx.PutValue("dump_hash.scrolled_docs", atomic.LoadInt64(&totalDocsScrolled))

					processedSize += docSize

					if global.Env().IsDebug {
						log.Debugf("[%v] slice[%v]:%v,%v-%v", processor.config.Elasticsearch, slice, docSize, processedSize, totalHits)
					}

					initScrollID = scrollID

					progress.IncreaseWithTotal(processor.config.Output, "dump-hash-"+util.ToString(tempSlice), docSize, int(totalHits))

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
	duration := time.Since(start)
	log.Infof("dump finished, es: %v, index: %v, docs: %v, duration: %vs, qps: %v ", processor.config.Elasticsearch, processor.config.Indices, totalDocsNeedToScroll, duration, math.Ceil(float64(totalDocsNeedToScroll)/math.Ceil((duration.Seconds()))))

	return nil
}

func (processor *DumpHashProcessor) processingDocs(data []byte, outputQueueName string) int {
	docSize := 0
	var docs = map[int]*bytebufferpool.ByteBuffer{}
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		id, err := jsonparser.GetString(value, "_id")
		if err != nil {
			panic(err)
		}

		source, _, _, err := jsonparser.Get(value, "_source")
		if err != nil {
			panic(err)
		}

		util.WalkBytesAndReplace(source, util.NEWLINE, util.SPACE)

		hash := processor.Hash(processor.config.HashFunc, source)

		partitionID := elastic.GetShardID(7, util.UnsafeStringToBytes(id), processor.config.PartitionSize)

		buffer, ok := docs[partitionID]
		if !ok {
			buffer = bytebufferpool.Get("dump_hash")
		}
		_, err = buffer.WriteBytesArray([]byte(id), []byte(","), hash)
		if err != nil {
			panic(err)
		}
		if processor.config.KeepSourceInResult {
			_, err = buffer.WriteBytesArray([]byte(","), source)
			if err != nil {
				panic(err)
			}
		}

		_, err = buffer.WriteBytesArray([]byte("\n"))
		if err != nil {
			panic(err)
		}
		docSize++

		docs[partitionID] = buffer

	}, "hits", "hits")

	for k, v := range docs {
		file := path.Join(global.Env().GetDataDir(), "diff", outputQueueName+"-"+util.ToString(k))
		processor.files.Store(file, true)
		handler := rotate.GetFileHandler(file, processor.config.RotateConfig)
		handler.Write(v.Bytes())
		bytebufferpool.Put("dump_hash", v)
	}

	stats.IncrementBy("scrolling_processing."+outputQueueName, "docs", int64(docSize))

	return docSize
}

func (processor *DumpHashProcessor) Release() error {
	var toRelease []string
	processor.files.Range(func(key, value any) bool {
		file, ok := key.(string)
		if !ok {
			log.Errorf("invalid file path")
			return true
		}
		toRelease = append(toRelease, file)
		return true
	})
	for _, file := range toRelease {
		rotate.ReleaseFileHandler(file)
	}
	return nil
}

func (processor *DumpHashProcessor) Hash(hashFunc string, data []byte) []byte {
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
