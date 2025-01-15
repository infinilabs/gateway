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

/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package elastic

import (
	"fmt"
	"github.com/OneOfOne/xxhash"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"github.com/savsgio/gotils/bytes"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Queue struct {
	Type      string                 `config:"type"`
	QueueName string                 `config:"queue_name"`
	Labels    map[string]interface{} `config:"labels,omitempty"`
}

type CommitConfig struct {
	QueueName      string `config:"queue_name" json:"queue_name,omitempty"`
	Group          string `config:"group" json:"group,omitempty"`
	Name           string `config:"name" json:"name,omitempty"`
	CommitInterval string `config:"interval"`
}

type BulkRequestResort struct {
	BatchSizeInDocs int `config:"batch_size_in_docs"` //batch size for each bulk request
	BatchSizeInMB   int `config:"batch_size_in_mb"`

	MinBufferSize           int    `config:"min_buffer_size"`
	MaxBufferSize           int    `config:"max_buffer_size"`
	MinDocPaddingForOneDocs int    `config:"min_doc_versions_for_one_doc"`
	IdleTimeoutInSeconds    string `config:"idle_timeout_in_seconds"`
	idleTimeout             time.Duration
	commitTimeout           time.Duration

	TagOnComplete string `config:"tag_on_complete"` //add tag to parent context when all documents are processed

	Elasticsearch string       `config:"elasticsearch"`
	PartitionSize int          `config:"partition_size"`
	OutputQueue   Queue        `config:"output_queue"`
	CommitConfig  CommitConfig `config:"commit_offset"`

	documentPartitionSorter sync.Map

	batchSizeInBytes    int
	inputQueueConfig    *queue.QueueConfig
	inputConsumerConfig *queue.ConsumerConfig
	bulkBufferPool      *elastic.BulkBufferPool
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_request_resort", NewBulkRequestResort, &BulkRequestResort{})
}

func NewBulkRequestResort(c *config.Config) (pipeline.Filter, error) {
	runner := BulkRequestResort{
		MinBufferSize:           10000,
		MaxBufferSize:           1000000,
		MinDocPaddingForOneDocs: 1000,
		BatchSizeInDocs:         5000,
		BatchSizeInMB:           10,
		IdleTimeoutInSeconds:    "10s",
		OutputQueue: Queue{
			Type:      "queue",
			QueueName: "sorted_docs",
			Labels: map[string]interface{}{
				"type": "bulk_request_resort",
			},
		},
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if runner.BatchSizeInMB <= 0 {
		runner.BatchSizeInMB = 10
	}
	runner.batchSizeInBytes = runner.BatchSizeInMB * 1024 * 1024

	if runner.Elasticsearch != "" {
		runner.OutputQueue.Labels["elasticsearch"] = runner.Elasticsearch
	}

	runner.bulkBufferPool = elastic.NewBulkBufferPool("bulk_request_resort", 1024*1024*1024, 100000)

	runner.idleTimeout = util.GetDurationOrDefault(runner.IdleTimeoutInSeconds, 10*time.Second)
	runner.commitTimeout = util.GetDurationOrDefault(runner.CommitConfig.CommitInterval, 10*time.Second)

	runner.documentPartitionSorter = sync.Map{}

	qConfig := queue.GetOrInitConfig(runner.CommitConfig.QueueName)
	cConfig := queue.GetOrInitConsumerConfig(qConfig.ID, runner.CommitConfig.Group, runner.CommitConfig.Name)

	runner.inputQueueConfig = qConfig
	runner.inputConsumerConfig = cConfig

	// for partitions
	for i := 0; i < runner.PartitionSize; i++ {

		sorter := &Sorter{filter: &runner}
		sorter.partitionID = i
		sorter.documentBuffer = runner.NewDocumentBuffer(i, runner.OutputQueue.QueueName, runner.MinBufferSize, runner.MaxBufferSize, runner.idleTimeout, runner.MinDocPaddingForOneDocs)

		runner.documentPartitionSorter.Store(i, sorter)

		go sorter.run()
	}

	//commit offset
	go func() {

		var lastCommittedOffset int64 = -1
		var lastCommittedTime = time.Now()

		for {
			var newlyEarlyCommit int64 = -1
			var newlyLatestCommit int64 = -1
			var haveDocs bool = false
			runner.documentPartitionSorter.Range(func(key, value interface{}) bool {
				sorter := value.(*Sorter)
				if newlyEarlyCommit == -1 {
					newlyEarlyCommit = sorter.lastCommittedOffset
				} else {
					if sorter.lastCommittedOffset < newlyEarlyCommit {
						newlyEarlyCommit = sorter.lastCommittedOffset
					}
				}
				if sorter.lastCommittedOffset > newlyLatestCommit {
					newlyLatestCommit = sorter.lastCommittedOffset
				}

				if sorter.documentBuffer.docsCount.Load() > 0 {
					haveDocs = true
				}

				return true
			})

			//log.Error("newlyEarlyCommit:",newlyEarlyCommit," newlyLatestCommit:",newlyLatestCommit," lastCommittedOffset:",lastCommittedOffset)

			if !haveDocs && util.Since(lastCommittedTime) > runner.commitTimeout {
				newlyEarlyCommit = newlyLatestCommit
			}

			if newlyEarlyCommit > lastCommittedOffset && newlyEarlyCommit > 0 {
				offset := queue.NewOffset(0, newlyEarlyCommit)
				_, err := queue.CommitOffset(runner.inputQueueConfig, runner.inputConsumerConfig, offset)
				log.Debug("commit offset :", runner.inputQueueConfig.Name, " -> ", offset)
				if err != nil {
					panic(err)
				}
				lastCommittedOffset = newlyEarlyCommit
				lastCommittedTime = time.Now()
			}

			time.Sleep(10 * time.Second)
			//log.Error("time.sleep 10s")
		}

	}()

	return &runner, nil
}

func (filter *BulkRequestResort) Name() string {
	return "bulk_request_resort"
}

func (filter *BulkRequestResort) getDocBuffer(partitionID int) *DocumentBuffer {
	sorter, ok := filter.documentPartitionSorter.Load(partitionID)
	if ok {
		return sorter.(*Sorter).documentBuffer
	}
	panic("invlid partition id")
}

func (filter *BulkRequestResort) Filter(ctx *fasthttp.RequestCtx) {

	//skip none bulk requests
	pathStr := util.UnsafeBytesToString(ctx.PhantomURI().Path())
	if !util.SuffixStr(pathStr, "/_bulk") {
		return
	}

	//process request and response

	//for bulk requests, we need to detect the conflicts
	//check response, if response success, then try to replicate the request
	//if the request have same id with higher version already exists, we need to resort the requests

	requestBody := ctx.Request.GetRawBody()
	responseBody := ctx.Response.GetRawBody()

	offsetStr := ctx.Request.Header.Peek("LAST_PRODUCED_MESSAGE_OFFSET")
	queueName, _ := ctx.GetString("MESSAGE_QUEUE_NAME")
	thisMessageOffset := ctx.Get("MESSAGE_OFFSET").(queue.Offset)
	nextMessageOffset := ctx.Get("NEXT_MESSAGE_OFFSET").(queue.Offset)

	replicaID := ctx.Request.Header.Peek("X-Replicated-ID")

	if responseBody != nil && requestBody != nil {
		var docsToReplicate = map[int]elastic.VersionInfo{}
		var docOffset = -1

		items, _, _, err := jsonparser.Get(responseBody, "items")
		if err == nil {
			jsonparser.ArrayEach(items, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

				docOffset += 1

				item, _, _, err := jsonparser.Get(value, "index")
				if err != nil {
					item, _, _, err = jsonparser.Get(value, "delete")
					if err != nil {
						item, _, _, err = jsonparser.Get(value, "create")
						if err != nil {
							item, _, _, err = jsonparser.Get(value, "update")
						}
					}
				}

				if err == nil {
					statusCode, err := jsonparser.GetInt(item, "status")
					if err != nil {
						panic(err)
					}
					if global.Env().IsDebug {
						stats.Increment("bulk_request_resort", fmt.Sprintf("%v_%v_parse", queueName, statusCode))
					}
					if statusCode > 300 {
						//log.Error(status,",",string(value))
						return
					}

					docId, err := jsonparser.GetString(item, "_id")
					if err != nil {
						panic(err)
					}

					result, _ := jsonparser.GetString(item, "result")
					if result == "noop" {
						//log.Error("skip noop result: ",string(result))
						return
					}

					//_version/_primary_term/_seq_no
					version := elastic.VersionInfo{}
					version.Version, _ = jsonparser.GetInt(item, "_version")
					version.PrimaryTerm, _ = jsonparser.GetInt(item, "_primary_term")
					version.SequenceNumber, _ = jsonparser.GetInt(item, "_seq_no")
					version.Index, _ = jsonparser.GetString(item, "_index")
					version.ID = docId
					version.DocOffset = docOffset
					version.MessageOffset = string(offsetStr)
					version.ThisMessageOffset = thisMessageOffset
					version.ReplicationID = string(replicaID)
					version.Path = fmt.Sprintf("%s/%s/%s", version.Index, "_doc", version.ID)
					version.Status = statusCode
					version.Time = util.GetLowPrecisionCurrentTime()

					docsToReplicate[docOffset] = version

					if version.Version == 0 {
						log.Error(string(value))
					}
				}
			})
		}

		//pick requests
		var offset = 0
		var collect = false
		var lastRecord elastic.VersionInfo

		docs := map[int][]elastic.VersionInfo{}

		elastic.WalkBulkRequests(pathStr, requestBody, nil,
			func(metaBytes []byte, actionStr, index, typeName, id, routing string, docCount int) (err error) {
				if lastRecord, collect = docsToReplicate[offset]; collect {
					lastRecord.Payload = append(lastRecord.Payload, bytes.Copy(metaBytes))
				}
				offset++
				return nil
			}, func(payloadBytes []byte, actionStr, index, typeName, id, routing string) {
				if collect {
					lastRecord.Payload = append(lastRecord.Payload, bytes.Copy(payloadBytes))
				}
			}, func(actionStr, index, typeName, id, routing string) {
				if collect {

					hash := xxHashPool.Get().(*xxhash.XXHash32)
					hash.Reset()
					hash.WriteString(id)

					partitionID := int(hash.Sum32()) % filter.PartitionSize
					docs[partitionID] = append(docs[partitionID], lastRecord)

					xxHashPool.Put(hash)
				}
			})

		if len(docs) > 0 {

			for partitionID, versions := range docs {

				if len(versions) > 1 {
					//resort again
					SortDocumentsByVersion(versions)
				}

				//add to buffer
				queueBuffer := filter.getDocBuffer(partitionID)
				queueBuffer.LastOffset = nextMessageOffset
				queueBuffer.Add(versions)
			}
		}

	} else {
		log.Error("invalid request")
	}

}

type Sorter struct {
	filter              *BulkRequestResort
	partitionID         int
	documentBuffer      *DocumentBuffer
	lastCommittedOffset int64
}

// 启动一个goroutine模拟持续读取文档
func (s *Sorter) run() {

	outputQueue := queue.AdvancedGetOrInitConfig(s.filter.OutputQueue.Type, fmt.Sprintf("%v#%v", s.filter.OutputQueue.QueueName, s.partitionID), s.filter.OutputQueue.Labels)

	log.Debug("init queue:", outputQueue.Name, " type:", outputQueue.Type, " partition:", s.partitionID, " labels:", s.filter.OutputQueue.Labels)

	producer, err := queue.AcquireProducer(outputQueue)
	defer func() {
		if producer != nil {
			producer.Close()
		}
	}()

	if err != nil {
		panic(err)
	}
	latestVersions := map[string]int64{}
	//count := 0

	var latestEarlyCommitOffset int64 = -1
	var lastCommitableOffset int64 = -1
	s.lastCommittedOffset = -1
	var lastCommitTime time.Time = util.GetLowPrecisionCurrentTime()
	var hit = false

	bulkBuffer := s.filter.bulkBufferPool.AcquireBulkBuffer()

	defer s.filter.bulkBufferPool.ReturnBulkBuffer(bulkBuffer)
	for {
		if bulkBuffer.GetMessageCount() == 0 {
			if global.ShuttingDown() {
				return
			}
		}

		documentBuffer := s.documentBuffer

		bulkBuffer.ResetData()
		bulkBuffer.Queue = s.filter.inputQueueConfig.ID

		for {

			// 如果缓冲区满足读取条件，进行读取
			docCount, documents := documentBuffer.GetDocuments(s.filter.BatchSizeInDocs)

			if docCount > 0 {

				//if global.Env().IsDebug {
				stats.IncrementBy("bulk_request_resort", fmt.Sprintf("%v_%v_get", s.filter.inputQueueConfig.ID, s.partitionID), int64(docCount))
				//}

				for _, doc := range documents {
					//SortDocumentsByVersion(docs)

					//count += len(docs)

					//for _, doc := range docs {

					offset := doc.ThisMessageOffset.Position
					if latestEarlyCommitOffset == -1 || offset < latestEarlyCommitOffset {
						latestEarlyCommitOffset = offset
					}

					if lastCommitableOffset == -1 || offset > lastCommitableOffset {
						lastCommitableOffset = offset
					}

					v, ok := latestVersions[doc.Path]
					if ok {
						if v >= doc.Version {
						} else {
							latestVersions[doc.Path] = doc.Version
						}
					} else {
						latestVersions[doc.Path] = doc.Version
					}

					//add to bulk buffer
					bulkBuffer.WriteMessageID(doc.Path)

					for _, b := range doc.Payload {
						bulkBuffer.WriteNewByteBufferLine("success", b)
					}
					//}
				}
				hit = true
			} else {
				time.Sleep(1 * time.Second)
				//log.Error("time.sleep 1s")
			}

			var mustCommitAndExit = false
			//handle final idle commit, no new message in queue
			if (util.Since(lastCommitTime) > s.filter.commitTimeout &&
				documentBuffer.docsCount.Load() == 0 &&
				documentBuffer.LastOffset.Position > s.lastCommittedOffset &&
				documentBuffer.LastOffset.Position > 0) || global.ShuttingDown() {

				lastCommitableOffset = documentBuffer.LastOffset.Position
				log.Info("all message processed, start commit offset:", documentBuffer.LastOffset, ",", lastCommitableOffset)

				mustCommitAndExit = true
			}

			//if it is ok to submit and commit
			if mustCommitAndExit || bulkBuffer.GetMessageCount() > 0 &&
				(bulkBuffer.GetMessageCount() > s.filter.BatchSizeInDocs ||
					bulkBuffer.GetMessageSize() > s.filter.batchSizeInBytes ||
					util.Since(lastCommitTime) > s.filter.commitTimeout) {

				if bulkBuffer.GetMessageCount() > 0 {
					requests := []queue.ProduceRequest{}
					requests = append(requests, queue.ProduceRequest{
						Topic: outputQueue.ID,
						Key:   []byte(bulkBuffer.Queue),
						Data:  bytes.Copy(bulkBuffer.GetMessageBytes()),
					})

					//log.Error("produce to output :", outputQueue.Name, ",pid:", s.partitionID, " -> ", len(requests), " -> ", bulkBuffer.GetMessageCount())
					_, err := producer.Produce(&requests)
					if err != nil {
						panic(err)
					}
					bulkBuffer.Reset()
				}

				if lastCommitableOffset > 0 && lastCommitableOffset > s.lastCommittedOffset {
					s.lastCommittedOffset = lastCommitableOffset
					lastCommitTime = util.GetLowPrecisionCurrentTime()
				}

				if mustCommitAndExit {
					break
				}

				if !hit {
					// 如果不满足读取条件，等待一段时间后再次检查
					time.Sleep(1 * time.Second)
					//log.Error("time.sleep 1s")

					hit = false
				}

			}

		}

		if !hit {
			// 如果不满足读取条件，等待一段时间后再次检查
			time.Sleep(1 * time.Second)
			//log.Error("time.sleep 1s")

			hit = false
		}
	}
}

//type Docs struct {
//	Docs []elastic.VersionInfo
//}

// DocumentBuffer 表示文档的缓冲区
type DocumentBuffer struct {
	//mu            sync.Mutex
	documents     chan elastic.VersionInfo //TODO，在最旧的 offset 里面，但是最近 docs 有更新，说明请求还在变化，可能还有更新的在路上
	maxBufferSize int
	minBufferSize int
	docsCount     atomic.Int64
	lastWriteTime time.Time
	//docs          sync.Map //map[string]*Docs //doc_id -> offset
	//writeBlocked  atomic.Bool
	idleTimeout           time.Duration
	minSafeDocPaddingSize int //min doc size to be send when latest doc is still fresh

	LastOffset queue.Offset

	lastToKeep []elastic.VersionInfo
}

// NewDocumentBuffer 创建一个新的文档缓冲区
func (filter *BulkRequestResort) NewDocumentBuffer(partitionID int, queueName string, minBufferSize, maxBufferSize int, idleTimeout time.Duration, minDocPaddingSize int) *DocumentBuffer {
	buffer := &DocumentBuffer{
		documents:             make(chan elastic.VersionInfo, maxBufferSize),
		minBufferSize:         minBufferSize,
		maxBufferSize:         maxBufferSize,
		docsCount:             atomic.Int64{},
		idleTimeout:           idleTimeout,
		minSafeDocPaddingSize: minDocPaddingSize,
		//docs:                  sync.Map{}, // make(map[string]*Docs),
		lastWriteTime: util.GetLowPrecisionCurrentTime(),
	}

	return buffer
}

// Add 将文档添加到缓冲区
func (b *DocumentBuffer) Add(docs []elastic.VersionInfo) {

RETRY:
	if b.docsCount.Load() > int64(b.maxBufferSize) {
		time.Sleep(1 * time.Second)
		//log.Error("time.sleep 1s")

		//log.Error("buffer full, drop docs")
		if !global.ShuttingDown() {
			goto RETRY
		}
	}

	//log.Error("start adding docs:",len(docs))

	b.docsCount.Add(int64(len(docs)))
	for _, v := range docs {
		b.documents <- v
	}

	//b.documents <- docs
	//log.Error("end adding docs:",len(docs))

	//b.mu.Lock()
	//defer b.mu.Unlock()
	//
	//for _, v := range docs {
	//	docs, ok := b.docs.LoadOrStore(v.Path, &Docs{Docs: []elastic.VersionInfo{v}})
	//	if ok {
	//		v1 := docs.(*Docs)
	//		v1.Docs = append(v1.Docs, v)
	//	} else {
	//		b.documents <- docs.(*Docs)
	//	}
	//}

	b.lastWriteTime = util.GetLowPrecisionCurrentTime()

}

// GetDocuments 返回最旧的文档通道，最多读取指定数量的文档
func (b *DocumentBuffer) GetDocuments(count int) (int, []elastic.VersionInfo) {
	predictDocs := int(b.docsCount.Load()) - count
	if predictDocs < b.minBufferSize {
		if util.Since(b.lastWriteTime) < b.idleTimeout {
			return 0, []elastic.VersionInfo{}
		}
		if b.docsCount.Load() == 0 {
			return 0, []elastic.VersionInfo{}
		}
	}

	var docsToCleanup = []elastic.VersionInfo{}

	//add last to keep
	docsToCleanup = append(docsToCleanup, b.lastToKeep...)

	//cleanup
	b.lastToKeep = []elastic.VersionInfo{}

	var docsCountToCleanup = 0
	for {
		select {
		case docs := <-b.documents:
			//doc := docs.Docs
			//if len(docs) > 0 {
			//b.docs.Delete(doc[0].Path) //DELETE map after popup, may unlock the map
			docsToCleanup = append(docsToCleanup, docs)
			docsCountToCleanup += 1 //len(docs)
			//}
			if docsCountToCleanup >= count {
				goto READ
			}
			break
		case <-time.After(10 * time.Second):
			// call timed out
			//return false
			goto READ
		}
	}

READ:

	//handle each docs
	//documentQueue := []elastic.VersionInfo{}
	removedDocs := 0
	//removedDocsGroup := 0
	docsToKeep := []elastic.VersionInfo{}

	//if docsCountToCleanup == 0 {
	//	return 0, []elastic.VersionInfo{}
	//}

	//log.Error("docs to clean up:", docsCountToCleanup, " of ", b.docsCount.Load(), ",", util.Since(b.lastWriteTime), ",count:", count, ",minBufferSize:", b.minBufferSize, ",maxBufferSize:", b.maxBufferSize, ",idleTimeout:", b.idleTimeout)

	////折叠相同 ID 的文档，只保留最新的文档
	//for _, doc := range docsToCleanup {
	//
	//	//如果一个 ID 的文档有很多更新，并且最新的文档还比较新鲜，那么我们保留最近的文档，将旧的文档消费走，新的继续保留最近的 N 个，比如 10 个
	//	if len(doc) > 1 {
	//		SortDocumentsByVersion(doc)
	//	}
	//
	//	//firstRecordTime := doc[0].Time
	//	lastRecordTime := doc[len(doc)-1].Time
	//
	//	var addToResult = false
	//	if len(doc) == 1 { // 当 buffer 满了，没有新的数据进来了
	//
	//		//现有的数据，也需要根据时间先后顺序排序之后再消费掉
	//		if int(b.docsCount.Load()) >= b.minBufferSize && util.Since(lastRecordTime) > b.idleTimeout {
	//			addToResult = true
	//		}
	//	}
	//
	//	if addToResult || int(b.docsCount.Load()) < b.minBufferSize && util.Since(lastRecordTime) > b.idleTimeout { //last record is not fresh
	//		removedDocs += len(doc)
	//		documentQueue = append(documentQueue, doc)
	//	} else {
	//		//TODO 有可能单个文档达不到最小安全文档大小，但是总文档数达到了最小安全文档大小，这种情况下，需要将文档全部消费掉
	//		if len(doc) > b.minSafeDocPaddingSize {
	//			newDocs := doc[len(doc)-b.minSafeDocPaddingSize:]
	//			docsToKeep = append(docsToKeep, newDocs)
	//			log.Error("1 adding back docs:", len(newDocs))
	//
	//			//remove old docs
	//			oldDocs := doc[:len(doc)-b.minSafeDocPaddingSize]
	//
	//			removedDocs += len(oldDocs) //all removedDocs processed
	//
	//			documentQueue = append(documentQueue, oldDocs)
	//
	//		} else if util.Since(lastRecordTime) > (b.idleTimeout) {
	//			keepSize := len(doc) / 2
	//			newDocs := doc[len(doc)-keepSize:]
	//			docsToKeep = append(docsToKeep, newDocs)
	//
	//			log.Error("2 adding back docs:", len(newDocs))
	//
	//			//remove old docs
	//			oldDocs := doc[:len(doc)-keepSize]
	//			removedDocs += len(oldDocs) //all removedDocs processed
	//
	//			log.Error("3 adding back docs:", len(oldDocs))
	//
	//			documentQueue = append(documentQueue, oldDocs)
	//		} else {
	//			log.Error("4 adding back docs:", len(doc),",",util.Since(lastRecordTime) > (b.idleTimeout),",",util.Since(lastRecordTime) )
	//			docsToKeep = append(docsToKeep, doc) //样本太小，直接丢回去
	//		}
	//	}
	//
	//	//remove temp docs, add back if it is necessary
	//	removedDocsGroup += 1 //per doc group with same id
	//}

	//Add back removedDocs to keep
	//log.Error("total adding back docs:", len(docsToKeep))

	//if len(docsToCleanup)>b.minBufferSize{
	SortDocumentsByTime(docsToCleanup)
	//}

	removedDocs = len(docsToCleanup)
	b.lastToKeep = docsToKeep
	//if len(docsToKeep) > 0 {
	//	for _, v := range docsToKeep {
	//
	//		if len(v) > 0 {
	//			b.documents <- v
	//		}
	//	}
	//}

	b.docsCount.Add(int64(removedDocs * -1))

	//log.Error("+docs to clean up:", removedDocs, " of ", b.docsCount.Load(), ",", util.Since(b.lastWriteTime), ",count:", count, ",minBufferSize:", b.minBufferSize, ",maxBufferSize:", b.maxBufferSize, ",idleTimeout:", b.idleTimeout)

	return removedDocs, docsToCleanup
}

// SortDocumentsByVersion 按照文档版本进行排序
func SortDocumentsByVersion(docs []elastic.VersionInfo) {
	sort.Slice(docs, func(i, j int) bool {

		// 按照版本升序排序
		return docs[i].Version < docs[j].Version

		//// 按照版本升序排序
		//return docs[i].Path < docs[j].Path &&
		//	docs[i].Version < docs[j].Version&&
		//	docs[i].Time.Unix() < docs[j].Time.Unix()
	})
}

func SortDocumentsByTime(docs []elastic.VersionInfo) {
	sort.Slice(docs, func(i, j int) bool {

		// 按照版本升序排序
		return docs[i].Time.Unix() < docs[j].Time.Unix()

		//// 按照版本升序排序
		//return docs[i].Path < docs[j].Path &&
		//	docs[i].Version < docs[j].Version&&
		//	docs[i].Time.Unix() < docs[j].Time.Unix()
	})
}
