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

package elastic

import (
	"fmt"
	"github.com/OneOfOne/xxhash"
	"runtime"
	"time"

	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"github.com/savsgio/gotils/bytes"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	config        *BulkReshuffleConfig
	esConfig      *elastic.ElasticsearchConfig
	docBufferPool *bytebufferpool.Pool
}

func (this *BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}

type Level string

const ClusterLevel = "cluster"
const NodeLevel = "node"
const IndexLevel = "index"
const ShardLevel = "shard"

var startPart = []byte("{\"took\":0,\"errors\":false,\"items\":[")
var itemPart = []byte("{\"index\":{\"_index\":\"fake-index\",\"_type\":\"doc\",\"_id\":\"1\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":1,\"successful\":1,\"failed\":0},\"_seq_no\":1,\"_primary_term\":1,\"status\":200}}")
var endPart = []byte("]}")

type BulkReshuffleConfig struct {
	TagsOnSuccess []string `config:"tag_on_success"`

	Elasticsearch          string `config:"elasticsearch"`
	QueuePrefix            string `config:"queue_name_prefix"`
	Level                  string `config:"level"` //cluster/node(will,change)/index/shard/partition
	PartitionSize          int    `config:"partition_size"`
	FixNullID              bool   `config:"fix_null_id"`
	ContinueAfterReshuffle bool   `config:"continue_after_reshuffle"`
	IndexStatsAnalysis     bool   `config:"index_stats_analysis"`
	ActionStatsAnalysis    bool   `config:"action_stats_analysis"`

	ContinueMetadataNotFound bool `config:"continue_metadata_missing"`

	ValidateRequest bool `config:"validate_request"`

	//split all lines into memory rather than scan
	ValidEachLine bool  `config:"validate_each_line"`
	ValidMetadata bool  `config:"validate_metadata"`
	ValidPayload  bool  `config:"validate_payload"`
	StickToNode   bool  `config:"stick_to_node"`
	EnabledShards []int `config:"shards"`

	BufferPoolEnabled bool   `config:"bytes_buffer_enabled"`
	MaxBufferCount    uint32 `config:"max_buffer_items"`
	MaxBufferSize     uint32 `config:"max_buffer_size"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_reshuffle",
		pipeline.FilterConfigChecked(NewBulkReshuffle, pipeline.RequireFields("elasticsearch")),
		&BulkReshuffleConfig{})
}

func NewBulkReshuffle(c *config.Config) (pipeline.Filter, error) {

	cfg := BulkReshuffleConfig{
		QueuePrefix:         "async_bulk",
		IndexStatsAnalysis:  true,
		ActionStatsAnalysis: true,
		FixNullID:           true,
		MaxBufferCount:      10000,
		MaxBufferSize:       1024 * 1024 * 1024,
		Level:               NodeLevel,
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := BulkReshuffle{config: &cfg}

	if cfg.Elasticsearch == "" {
		panic(errors.New("elasticsearch is required"))
	}

	runner.esConfig = elastic.GetConfig(cfg.Elasticsearch)

	if cfg.BufferPoolEnabled {
		runner.docBufferPool = bytebufferpool.NewTaggedPool("bulk_reshuffle_request_docs"+util.GetUUID(), 0, cfg.MaxBufferSize, cfg.MaxBufferCount)
	}

	return &runner, nil
}

func (this *BulkReshuffle) Filter(ctx *fasthttp.RequestCtx) {

	pathStr := util.UnsafeBytesToString(ctx.PhantomURI().Path())

	//拆解 bulk 请求，重新封装
	if util.SuffixStr(pathStr, "/_bulk") {

		ctx.Set(common.CACHEABLE, false)

		metadata := elastic.GetOrInitMetadata(this.esConfig)

		if metadata == nil {
			if rate.GetRateLimiter("cluster_metadata", this.config.Elasticsearch, 1, 1, 5*time.Second).Allow() {
				log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle", this.config.Elasticsearch)
			}
			return
		}

		body := ctx.Request.GetRawBody()

		var indexStatsData map[string]int
		var actionStatsData map[string]int

		//index-shardID -> buffer
		docBuf := map[string]*bytebufferpool.ByteBuffer{}

		validEachLine := this.config.ValidEachLine
		validMetadata := this.config.ValidMetadata
		validPayload := this.config.ValidPayload
		reshuffleType := this.config.Level
		fixNullID := this.config.FixNullID

		var buff *bytebufferpool.ByteBuffer
		var queueConfig *queue.QueueConfig
		indexAnalysis := this.config.IndexStatsAnalysis   //sync and async
		actionAnalysis := this.config.ActionStatsAnalysis //sync and async
		validateRequest := this.config.ValidateRequest
		var collectedMeta bool //may skip request collect during metadata process
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
					log.Error("error in bulk_reshuffle,", v)
				}
				if this.config.BufferPoolEnabled {
					for _, y := range docBuf {
						this.docBufferPool.Put(y)
					}
				}
			}
		}()

		var hitMetadataNotFound bool

		docCount, err := elastic.WalkBulkRequests(pathStr, body, func(eachLine []byte) (skipNextLine bool) {
			if validEachLine {
				obj := map[string]interface{}{}
				err := util.FromJSONBytes(eachLine, &obj)
				if err != nil {
					log.Error("error on validate scannedByte:", string(eachLine))
					panic(err)
				}
			}
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id, routing string, offset int) (err error) {

			collectedMeta = false

			metaStr := util.UnsafeBytesToString(metaBytes)

			shardID := 0

			var idNew string

			if (actionStr == elastic.ActionIndex || actionStr == elastic.ActionCreate) && (len(id) == 0 || id == "null") && fixNullID {
				id = util.GetUUID()
				idNew = id
				if global.Env().IsDebug {
					log.Trace("generated ID,", id, ",", metaStr)
				}
			}

			if idNew != "" {
				var err error
				metaBytes, err = elastic.UpdateBulkMetadata(actionStr, metaBytes, "", "", idNew)
				if err != nil {
					panic(err)
				}
				if global.Env().IsDebug {
					log.Trace("updated meta,", id, ",", metaStr)
				}
			}

			if indexAnalysis {
				//init
				if indexStatsData == nil {
					if indexStatsData == nil {
						indexStatsData = map[string]int{}
					}
				}

				//stats
				indexName := elastic.RemoveDotFromIndexName(index, "#")
				v, ok := indexStatsData[indexName]
				if !ok {
					indexStatsData[indexName] = 1
				} else {
					indexStatsData[indexName] = v + 1
				}
			}

			if actionAnalysis {
				//init
				if actionStatsData == nil {
					if actionStatsData == nil {
						actionStatsData = map[string]int{}
					}
				}

				//stats
				v, ok := actionStatsData[actionStr]
				if !ok {
					actionStatsData[actionStr] = 1
				} else {
					actionStatsData[actionStr] = v + 1
				}
			}

			if validMetadata {
				obj := map[string]interface{}{}
				err := util.FromJSONBytes(metaBytes, &obj)
				if err != nil {
					log.Error("error on validate action metadata")
					panic(err)
				}
			}

			if actionStr == "" || index == "" || id == "" {
				log.Warn("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", metaStr)
				panic(errors.Error("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", metaStr))
			}

			var nodeID, ShardIDStr string

			if reshuffleType == NodeLevel || reshuffleType == ShardLevel {
				//get routing table of index
				table, err := metadata.GetIndexRoutingTable(index)
				if err != nil {
					if rate.GetRateLimiter("index_routing_table_not_found", index, 1, 2, time.Minute*1).Allow() {
						log.Warn(index, ",", metaStr, ",", err)
					}
					if this.config.ContinueMetadataNotFound {
						hitMetadataNotFound = true
						return nil
					}
					panic(err)
				} else {
					//check if it is not only one shard
					totalShards := len(table)
					if totalShards > 1 {
						//如果 shards=1，则直接找主分片所在节点，否则计算一下。
						shardID = elastic.GetShardID(metadata.GetMajorVersion(), []byte(id), totalShards)

						if global.Env().IsDebug {
							log.Tracef("%s/%s => %v", index, id, shardID)
						}

						//check enabled shards
						if len(this.config.EnabledShards) > 0 {
							if !util.ContainsInAnyInt32Array(shardID, this.config.EnabledShards) {
								log.Debugf("shard %v-%v not enabled, skip processing", index, shardID)
								return nil
							}
						}
					}

					ShardIDStr = util.IntToString(shardID)
					shardInfo, err := metadata.GetPrimaryShardInfo(index, ShardIDStr)
					if err != nil || shardInfo == nil {
						if rate.GetRateLimiter(fmt.Sprintf("shard_info_not_found_%v", index), ShardIDStr, 1, 5, time.Minute*1).Allow() {
							log.Warn("shardInfo was not found,", index, ",", shardID)
						}

						if this.config.ContinueMetadataNotFound {
							hitMetadataNotFound = true
							return nil
						}
						panic(errors.Error("shard info was not found,", index, ",", shardID, ",", err))
					}
					nodeID = shardInfo.Node
				}
			}

			//partition
			var partitionID int
			var partitionSuffix = ""

			//get queue config

			//reset queueConfig
			queueConfig = nil
			queueKey := "" //TODO

			if this.config.PartitionSize > 1 {
				if reshuffleType != ShardLevel {
					xxHash := xxHashPool.Get().(*xxhash.XXHash32)
					xxHash.Reset()
					xxHash.WriteString(id)
					partitionID = int(xxHash.Sum32()) % this.config.PartitionSize
					xxHashPool.Put(xxHash)
				} else {
					partitionID = elastic.GetShardID(metadata.GetMajorVersion(), []byte(id), this.config.PartitionSize)
				}
				partitionSuffix = "##" + util.IntToString(partitionID)
			}

			switch reshuffleType {
			case ClusterLevel:
				queueKey = this.config.QueuePrefix + "##cluster##" + this.esConfig.ID + partitionSuffix
				break
			case NodeLevel:
				if nodeID == "" {
					nodeID = "UNASSIGNED"
				}
				queueKey = this.config.QueuePrefix + "##node##" + this.esConfig.ID + "##" + nodeID + partitionSuffix
				break
			case IndexLevel:
				queueKey = this.config.QueuePrefix + "##index##" + this.esConfig.ID + "##" + index + partitionSuffix
				break
			case ShardLevel:
				if ShardIDStr == "" {
					ShardIDStr = "UNASSIGNED"
				}
				queueKey = this.config.QueuePrefix + "##shard##" + this.esConfig.ID + "##" + index + "##" + ShardIDStr + partitionSuffix
				break
			}

			if queueKey == "" {
				panic(errors.Error("queue key can't be nil"))
			}

			var skipInit = false
			cfg1, ok := queue.SmartGetConfig(queueKey)
			if ok && len(cfg1.Labels) > 0 {
				_, ok := cfg1.Labels["type"] //check label bulk_reshuffle exists
				if ok {
					queueConfig = cfg1
					skipInit = true
				}
			}

			if !skipInit {
				//create new queue config
				labels := map[string]interface{}{}
				labels["type"] = "bulk_reshuffle" //type must have
				labels["level"] = reshuffleType
				labels["elasticsearch"] = this.esConfig.ID

				if this.config.PartitionSize > 1 {
					labels["partition_size"] = this.config.PartitionSize
					labels["partition"] = partitionID
				}

				//注册队列到元数据中，消费者自动订阅该队列列表，并根据元数据来分别进行相应的处理
				switch reshuffleType {
				case ClusterLevel:
					break
				case NodeLevel:
					labels["node_id"] = nodeID //need metadata
					break
				case IndexLevel:
					labels["index"] = index
					break
				case ShardLevel:
					labels["index"] = index
					labels["shard"] = shardID //need metadata
					break
				}

				queueConfig = queue.AdvancedGetOrInitConfig("", queueKey, labels)
				if queueConfig == nil {
					panic(errors.Error("queue config can't be nil"))
				}
			}

			if global.Env().IsDebug {
				log.Tracef("compute shard_id: %s/%s/%s => %v , %v", index, typeName, id, shardID, queueConfig)
			}

			////update actionItem
			buff, ok = docBuf[queueConfig.Name]
			if !ok {
				if this.config.BufferPoolEnabled {
					buff = this.docBufferPool.Get()
				} else {
					buff = &bytebufferpool.ByteBuffer{}
				}
				docBuf[queueConfig.Name] = buff
			}

			//add to major buffer
			elastic.SafetyAddNewlineBetweenData(buff, metaBytes)
			collectedMeta = true

			return nil
		}, func(payloadBytes []byte, actionStr, index, typeName, id, routing string) {

			//only if metadata is collected, than we collect payload, payload can't live without metadata
			if collectedMeta {
				buff, ok := docBuf[queueConfig.Name]
				if !ok {
					if this.config.BufferPoolEnabled {
						buff = this.docBufferPool.Get()
					} else {
						buff = &bytebufferpool.ByteBuffer{}
					}
					docBuf[queueConfig.Name] = buff
				}

				if global.Env().IsDebug {
					log.Trace("payload:", string(payloadBytes))
				}

				if payloadBytes != nil && len(payloadBytes) > 0 {

					elastic.SafetyAddNewlineBetweenData(buff, payloadBytes)

					if validPayload {
						obj := map[string]interface{}{}
						err := util.FromJSONBytes(payloadBytes, &obj)
						if err != nil {
							log.Error("error on validate action payload:", string(payloadBytes))
							panic(err)
						}
					}
				}
			}
		}, nil)

		if err != nil {
			if global.Env().IsDebug {
				log.Error(err)
			}
			panic(err)
		}

		//stats
		if indexAnalysis {
			ctx.Set("bulk_index_stats", indexStatsData)
			for k, v := range indexStatsData {
				//统计索引次数
				stats.IncrementBy("elasticsearch."+this.config.Elasticsearch+".indices", elastic.RemoveDotFromIndexName(k, "#"), int64(v))
			}
		}
		if actionAnalysis {
			ctx.Set("bulk_action_stats", actionStatsData)
			for k, v := range actionStatsData {
				//统计操作次数
				stats.IncrementBy("elasticsearch."+this.config.Elasticsearch+".operations", elastic.RemoveDotFromIndexName(k, "#"), int64(v))
			}
		}

		if ctx.Has("elastic_cluster_name") {
			es1 := ctx.MustGetStringArray("elastic_cluster_name")
			ctx.Set("elastic_cluster_name", append(es1, this.config.Elasticsearch))
		} else {
			ctx.Set("elastic_cluster_name", []string{this.config.Elasticsearch})
		}

		//skip async or not
		if this.config.ContinueMetadataNotFound && hitMetadataNotFound {
			if rate.GetRateLimiterPerSecond("metadata_not_found", "reshuffle", 1).Allow() {
				log.Debug("metadata not found, skip reshuffle")
			}
			return
		}

		//send to queue
		for x, y := range docBuf {

			if y.Len() <= 0 {
				log.Trace("empty doc buffer, skip processing: ", x)
				continue
			}

			if !util.BytesHasSuffix(y.B, elastic.NEWLINEBYTES) {
				y.Write(elastic.NEWLINEBYTES)
			}

			data := y.Bytes()

			if validateRequest {
				elastic.ValidateBulkRequest("aync-bulk", string(data))
			}

			if len(data) > 0 {

				cfg := queue.GetOrInitConfig(x)
				err := queue.Push(cfg, bytes.Copy(data))
				if err != nil {
					panic(err)
				}
				ctx.SetDestination(fmt.Sprintf("%v:%v", "queue", x))
			} else {
				log.Warn("zero message,", x, ",", len(data), ",", string(body))
			}
			if this.config.BufferPoolEnabled {
				this.docBufferPool.Put(y)
			}
		}

		//fake results
		ctx.SetContentType(JSON_CONTENT_TYPE)

		buffer := bytebufferpool.Get("fake_bulk_results")

		buffer.Write(startPart)
		for i := 0; i < docCount; i++ {
			if i != 0 {
				buffer.Write([]byte(","))
			}
			buffer.Write(itemPart)
		}
		buffer.Write(endPart)

		ctx.Response.SetBody(bytes.Copy(buffer.Bytes()))
		bytebufferpool.Put("fake_bulk_results", buffer)

		if len(this.config.TagsOnSuccess) > 0 {
			ctx.UpdateTags(this.config.TagsOnSuccess, nil)
		}

		ctx.Response.Header.Set("X-Async-Bulk", "true")
		ctx.Response.Header.Set("X-Bulk-Reshuffled", "true")

		if !this.config.ContinueAfterReshuffle {
			ctx.Response.SetStatusCode(200)
			ctx.Finished()
		}
	}

	//排除条件，非 _ 开头的索引。
	//可以指定排除和允许的索引，设置匹配的索引名称，通配符。

	//PUT/POST index/_doc/UUID
	//只有匹配到是单独的索引请求才会进行合并处理。
	//放内存里面，按节点或者分片为单位进行缓存，或者固定的通道数，固定通道数<按节点<按分片。

	//count、size 和 timeout 任意满足即进行 bulk 提交。
	//通过 ID 获取到分片所在节点位置，没有 ID 就获取到包含主分片的节点，均衡选择，或者主动生成 ID。
	//变成 bulk 格式

}

func batchUpdateJson(scannedByte []byte, action string, set, del map[string]string) (newBytes []byte, err error) {

	for k, _ := range del {
		scannedByte = jsonparser.Delete(scannedByte, action, k)
	}

	for k, v := range set {
		scannedByte, err = jsonparser.Set(scannedByte, []byte("\""+v+"\""), action, k)
		if err != nil {
			return scannedByte, err
		}
	}

	return scannedByte, err
}
