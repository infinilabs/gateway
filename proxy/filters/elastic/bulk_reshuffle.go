package elastic

import (
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
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
	"github.com/savsgio/gotils/bytes"
	"time"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	config *BulkReshuffleConfig
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
	TagsOnSuccess []string `config:"tag_on_success"` //hit limiter then add tag

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
	SafetyParse   bool  `config:"safety_parse"`
	ValidEachLine bool  `config:"validate_each_line"`
	ValidMetadata bool  `config:"validate_metadata"`
	ValidPayload  bool  `config:"validate_payload"`
	StickToNode   bool  `config:"stick_to_node"`
	DocBufferSize int   `config:"doc_buffer_size"`
	EnabledShards []int `config:"shards"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_reshuffle",
		pipeline.FilterConfigChecked(NewBulkReshuffle, pipeline.RequireFields("elasticsearch")),
		&BulkReshuffleConfig{})
}

func NewBulkReshuffle(c *config.Config) (pipeline.Filter, error) {

	cfg := BulkReshuffleConfig{
		DocBufferSize:       256 * 1024,
		QueuePrefix:         "async_bulk",
		IndexStatsAnalysis:  true,
		SafetyParse:         true,
		ActionStatsAnalysis: true,
		FixNullID:           true,
		Level:               NodeLevel,
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := BulkReshuffle{config: &cfg}

	return &runner, nil
}

var docBufferPool=  bytebufferpool.NewTaggedPool("bulk_request_docs",0,1024*1024*100,1000000)

func (this *BulkReshuffle) Filter(ctx *fasthttp.RequestCtx) {

	pathStr := util.UnsafeBytesToString(ctx.URI().Path())

	//拆解 bulk 请求，重新封装
	if util.SuffixStr(pathStr, "/_bulk") {

		ctx.Set(common.CACHEABLE, false)

		clusterName := this.config.Elasticsearch

		esConfig := elastic.GetConfig(clusterName)

		metadata := elastic.GetOrInitMetadata(esConfig)

		if metadata == nil {
			if rate.GetRateLimiter("cluster_metadata", clusterName, 1, 1, 5*time.Second).Allow() {
				log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle", clusterName)
			}
			//time.Sleep(1 * time.Second)
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
		var ok bool
		var queueConfig *queue.QueueConfig
		//var queueName string
		indexAnalysis := this.config.IndexStatsAnalysis   //sync and async
		actionAnalysis := this.config.ActionStatsAnalysis //sync and async
		validateRequest := this.config.ValidateRequest
		actionMeta := bytebufferpool.Get("bulk_request_action")
		defer bytebufferpool.Put("bulk_request_action", actionMeta)

		var docBuffer []byte
		docBuffer = elastic.BulkDocBuffer.Get(this.config.DocBufferSize) //doc buffer for bytes scanner
		defer elastic.BulkDocBuffer.Put(docBuffer)

		docCount, err := elastic.WalkBulkRequests(this.config.SafetyParse, body, docBuffer, func(eachLine []byte) (skipNextLine bool) {
			if validEachLine {
				obj := map[string]interface{}{}
				err := util.FromJSONBytes(eachLine, &obj)
				if err != nil {
					log.Error("error on validate scannedByte:", string(eachLine))
					panic(err)
				}
			}
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id,routing string) (err error) {

			metaStr := util.UnsafeBytesToString(metaBytes)

			shardID := 0

			//url level
			var urlLevelIndex string
			var urlLevelType string

			urlLevelIndex, urlLevelType = elastic.ParseUrlLevelBulkMeta(pathStr)

			var indexNew, typeNew, idNew string
			if index == "" && urlLevelIndex != "" {
				index = urlLevelIndex
				indexNew = urlLevelIndex
			}

			if typeName == "" && urlLevelType != "" {
				typeName = urlLevelType
				typeNew = urlLevelType
			}

			if (actionStr == elastic.ActionIndex || actionStr == elastic.ActionCreate) && (len(id) == 0 || id == "null") && fixNullID {
				id = util.GetUUID()
				idNew = id
				if global.Env().IsDebug {
					log.Trace("generated ID,", id, ",", metaStr)
				}
			}

			if indexNew != "" || typeNew != "" || idNew != "" {
				var err error
				metaBytes, err = updateJsonWithNewIndex(actionStr, metaBytes, indexNew, typeNew, idNew)
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
				v, ok := indexStatsData[index]
				if !ok {
					indexStatsData[index] = 1
				} else {
					indexStatsData[index] = v + 1
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
				return errors.Error("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", metaStr)
			}

			var nodeID, ShardIDStr string

			if reshuffleType == NodeLevel || reshuffleType == ShardLevel {
				//get routing table of index
				table, err := metadata.GetIndexRoutingTable(index)
				if err != nil {
					if rate.GetRateLimiter("index_routing_table_not_found", index, 1, 2, time.Minute*1).Allow() {
						log.Warn(index, ",", metaStr, ",", err)
					}
					return err
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
								return errors.Errorf("shard %s-%v not enabled, skip processing", index, shardID)
							}
						}
					}

					ShardIDStr = util.IntToString(shardID)
					shardInfo, err := metadata.GetPrimaryShardInfo(index, ShardIDStr)
					if err != nil {
						return errors.Error("shard info was not found,", index, ",", shardID, ",", err)
					}

					if shardInfo == nil {
						if rate.GetRateLimiter(fmt.Sprintf("shard_info_not_found_%v", index), ShardIDStr, 1, 5, time.Minute*1).Allow() {
							log.Warn("shardInfo was not found,", index, ",", shardID)
						}
						return errors.Error("shard info was not found,", index, ",", shardID)
					}
					nodeID = shardInfo.Node
				}
			}

			queueConfig = &queue.QueueConfig{}
			queueConfig.Source = "dynamic"
			queueConfig.Labels = util.MapStr{}
			queueConfig.Labels["type"] = "bulk_reshuffle"
			queueConfig.Labels["level"] = reshuffleType
			queueConfig.Labels["elasticsearch"] = esConfig.ID

			//partition
			var partitionSuffix = ""
			if this.config.PartitionSize > 1 {
				queueConfig.Labels["partition_size"] = this.config.PartitionSize
				partitionID := elastic.GetShardID(metadata.GetMajorVersion(), []byte(id), this.config.PartitionSize)
				queueConfig.Labels["partition"] = partitionID
				partitionSuffix = "##" + util.IntToString(partitionID)
			}

			//注册队列到元数据中，消费者自动订阅该队列列表，并根据元数据来分别进行相应的处理
			switch reshuffleType {
			case ClusterLevel:
				queueConfig.Name = this.config.QueuePrefix + "##cluster##" + esConfig.ID + partitionSuffix
				break
			case NodeLevel:

				if nodeID == "" {
					nodeID = "UNASSIGNED"
				}

				queueConfig.Labels["node_id"] = nodeID //need metadata
				queueConfig.Name = this.config.QueuePrefix + "##node##" + esConfig.ID + "##" + nodeID + partitionSuffix
				break
			case IndexLevel:
				queueConfig.Labels["index"] = index
				queueConfig.Name = this.config.QueuePrefix + "##index##" + esConfig.ID + "##" + index + partitionSuffix
				break
			case ShardLevel:

				if ShardIDStr == "" {
					ShardIDStr = "UNASSIGNED"
				}

				queueConfig.Labels["index"] = index
				queueConfig.Labels["shard"] = shardID //need metadata
				queueConfig.Name = this.config.QueuePrefix + "##shard##" + esConfig.ID + "##" + index + "##" + ShardIDStr + partitionSuffix
				break
			}

			if global.Env().IsDebug {
				log.Tracef("%s/%s/%s => %v , %v", index, typeName, id, shardID, queueConfig)
			}

			////update actionItem
			buff, ok = docBuf[queueConfig.Name]
			if !ok {
				buff = docBufferPool.Get()
				docBuf[queueConfig.Name] = buff
				var exists bool
				exists, err = queue.RegisterConfig(queueConfig.Name, queueConfig)
				if !exists && err != nil {
					panic(err)
				}
			}

			//保存临时变量
			actionMeta.Write(metaBytes)

			return nil
		}, func(payloadBytes []byte) {

			if actionMeta.Len() > 0 {
				buff, ok := docBuf[queueConfig.Name]
				if !ok {
					buff = docBufferPool.Get()
					docBuf[queueConfig.Name] = buff
				}

				if global.Env().IsDebug {
					log.Trace("metadata:", string(payloadBytes))
				}

				if buff.Len() > 0 {
					buff.Write(elastic.NEWLINEBYTES)
				}

				buff.Write(actionMeta.Bytes())
				if payloadBytes != nil && len(payloadBytes) > 0 {
					buff.Write(elastic.NEWLINEBYTES)
					buff.Write(payloadBytes)

					if validPayload {
						obj := map[string]interface{}{}
						err := util.FromJSONBytes(payloadBytes, &obj)
						if err != nil {
							log.Error("error on validate action payload:", string(payloadBytes))
							panic(err)
						}
					}
				}

				//cleanup actionMeta
				actionMeta.Reset()
			}
		})

		if err != nil {
			if global.Env().IsDebug {
				log.Error(err)
			}
			return
		}

		//send to queue
		for x, y := range docBuf {
			y.Write(elastic.NEWLINEBYTES)
			data := y.Bytes()

			if validateRequest {
				elastic.ValidateBulkRequest("aync-bulk", string(data))
			}

			if len(data) > 0 {

				cfg, ok := queue.SmartGetConfig(x)
				if !ok {
					panic(errors.Errorf("queue config [%v] not exists", x))
				}

				err := queue.Push(cfg, bytes.Copy(data))
				if err != nil {
					panic(err)
				}
				ctx.SetDestination(fmt.Sprintf("%v:%v", "async", x))
			} else {
				log.Warn("zero message,", x, ",", len(data), ",", string(body))
			}
			docBufferPool.Put(y)
		}

		if indexAnalysis {
			ctx.Set("bulk_index_stats", indexStatsData)
			for k, v := range indexStatsData {
				//统计索引次数
				stats.IncrementBy("elasticsearch."+clusterName+".indices", k, int64(v))
			}
		}
		if actionAnalysis {
			ctx.Set("bulk_action_stats", actionStatsData)
			for k, v := range actionStatsData {
				//统计操作次数
				stats.IncrementBy("elasticsearch."+clusterName+".operations", k, int64(v))
			}
		}

		if ctx.Has("elastic_cluster_name") {
			es1 := ctx.MustGetStringArray("elastic_cluster_name")
			ctx.Set("elastic_cluster_name", append(es1, clusterName))
		} else {
			ctx.Set("elastic_cluster_name", []string{clusterName})
		}

		//fake results
		ctx.SetContentType(JSON_CONTENT_TYPE)

		buffer := docBufferPool.Get()

		buffer.Write(startPart)
		for i := 0; i < docCount; i++ {
			if i != 0 {
				buffer.Write([]byte(","))
			}
			buffer.Write(itemPart)
		}
		buffer.Write(endPart)

		ctx.Response.AppendBody(bytes.Copy(buffer.Bytes()))
		docBufferPool.Put(buffer)

		if len(this.config.TagsOnSuccess) > 0 {
			ctx.UpdateTags(this.config.TagsOnSuccess, nil)
		}

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

	//newBytes = make([]byte, len(scannedByte))
	//copy(newBytes, scannedByte)

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

func updateJsonWithNewIndex(action string, scannedByte []byte, index, typeName, id string) (newBytes []byte, err error) {

	if global.Env().IsDebug {
		log.Trace("update:", action, ",", index, ",", typeName, ",", id)
	}

	newBytes = make([]byte, len(scannedByte))
	copy(newBytes, scannedByte)

	if index != "" {
		newBytes, err = jsonparser.Set(newBytes, []byte("\""+index+"\""), action, "_index")
		if err != nil {
			return newBytes, err
		}
	}
	if typeName != "" {
		newBytes, err = jsonparser.Set(newBytes, []byte("\""+typeName+"\""), action, "_type")
		if err != nil {
			return newBytes, err
		}
	}
	if id != "" {
		newBytes, err = jsonparser.Set(newBytes, []byte("\""+id+"\""), action, "_id")
		if err != nil {
			return newBytes, err
		}
	}

	return newBytes, err
}
