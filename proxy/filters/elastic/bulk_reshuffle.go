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
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"path"
	"time"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	config        *BulkReshuffleConfig
	bulkProcessor *BulkProcessor
}

func (this *BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}

type BulkReshuffleConfig struct {
	Elasticsearch       string `config:"elasticsearch"`
	Level               string `config:"level"`
	Mode                string `config:"mode"`
	FixNullID           bool   `config:"fix_null_id"`
	IndexStatsAnalysis  bool   `config:"index_stats_analysis"`
	ActionStatsAnalysis bool   `config:"action_stats_analysis"`

	ValidateRequest bool `config:"validate_request"`

	//split all lines into memory rather than scan
	SafetyParse bool `config:"safety_parse"`
	ValidEachLine   bool `config:"validate_each_line"`
	ValidMetadata   bool `config:"validate_metadata"`
	ValidPayload    bool `config:"validate_payload"`

	DocBufferSize       int                 `config:"doc_buffer_size"`
	Shards              []int               `config:"shards"`
	RotateConfig        rotate.RotateConfig `config:"rotate"`
	BulkProcessorConfig BulkProcessorConfig `config:"bulk_processing"`
}

func NewBulkReshuffle(c *config.Config) (pipeline.Filter, error) {

	cfg := BulkReshuffleConfig{
		DocBufferSize:       256 * 1024,
		IndexStatsAnalysis:  true,
		SafetyParse:  true,
		ActionStatsAnalysis: true,
		FixNullID:           true,
		Level:               "node",
		Mode:                "sync",
		RotateConfig: rotate.DefaultConfig,
		BulkProcessorConfig: DefaultBulkProcessorConfig,
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := BulkReshuffle{config: &cfg}

	runner.bulkProcessor = &BulkProcessor{
		RotateConfig: cfg.RotateConfig,
		Config:       cfg.BulkProcessorConfig,
	}
	return &runner, nil
}

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
			time.Sleep(10 * time.Second)
			return
		}

		body := ctx.Request.GetRawBody()

		var indexStatsData map[string]int
		var actionStatsData map[string]int

		//index-shardID -> buffer
		docBuf := map[string]*bytebufferpool.ByteBuffer{}
		buffHosts := map[string]string{}

		validEachLine := this.config.ValidEachLine
		validMetadata := this.config.ValidMetadata
		validPayload := this.config.ValidPayload
		reshuffleType := this.config.Level
		fixNullID := this.config.FixNullID

		//renameMapping, resolveIndexRename := this.GetStringMap("index_rename")

		var buff *bytebufferpool.ByteBuffer
		var ok bool
		var bufferKey string
		indexAnalysis := this.config.IndexStatsAnalysis   //sync and async
		actionAnalysis := this.config.ActionStatsAnalysis //sync and async
		validateRequest := this.config.ValidateRequest
		actionMeta := smallSizedPool.Get()
		defer smallSizedPool.Put(actionMeta)

		var docBuffer []byte
		docBuffer = p.Get(this.config.DocBufferSize) //doc buffer for bytes scanner
		defer p.Put(docBuffer)

		docCount, err := WalkBulkRequests(this.config.SafetyParse,body, docBuffer, func(eachLine []byte) (skipNextLine bool) {
			if validEachLine {
				obj := map[string]interface{}{}
				err := util.FromJSONBytes(eachLine, &obj)
				if err != nil {
					log.Error("error on validate scannedByte:", string(eachLine))
					panic(err)
				}
			}
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {

			metaStr := util.UnsafeBytesToString(metaBytes)

			shardID := 0

			//url level
			var urlLevelIndex string
			var urlLevelType string

			urlLevelIndex, urlLevelType = getUrlLevelBulkMeta(pathStr)

			var indexNew, typeNew, idNew string
			if index == "" && urlLevelIndex != "" {
				index = urlLevelIndex
				indexNew = urlLevelIndex
			}

			if typeName == "" && urlLevelType != "" {
				typeName = urlLevelType
				typeNew = urlLevelType
			}

			if (actionStr == actionIndex || actionStr == actionCreate) && len(id) == 0 && fixNullID {
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


			//check alias, if alias exists, update index to real index, for getting settings only, metadata still go alias
			//TODO


			//get routing table of index
			table,err:=metadata.GetIndexRoutingTable(index)
			if err!=nil{
				if rate.GetRateLimiter("index_routing_table_not_found", index, 1, 2, time.Minute*1).Allow() {
					log.Warn("index routing table not found,", index, ",", metaStr,",",err)
				}
				return err
			}

			//var indexSettings *elastic.IndexInfo
			//index,indexSettings,err=metadata.GetIndexSetting(index)
			//
			//if err!=nil{
			//	if rate.GetRateLimiter("index_setting_not_found", index, 1, 2, time.Minute*1).Allow() {
			//		log.Warn("index setting not found,", index, ",", metaStr,",",err)
			//	}
			//	return err
			//}

			//if indexSettings.Shards <= 0 || indexSettings.Status == "close" {
			//	log.Debugf("index %v closed,", indexSettings.Index)
			//	return errors.Errorf("index %v closed,", indexSettings.Index)
			//}

			//not only one shard
			totalShards:=len(table)
			if totalShards >1 {
				//如果 shards=1，则直接找主分片所在节点，否则计算一下。
				shardID = elastic.GetShardID(metadata.GetMajorVersion(), []byte(id), totalShards)

				if global.Env().IsDebug {
					log.Tracef("%s/%s => %v", index, id, shardID)
				}

				//skip configured shards
				if len(this.config.Shards) > 0 {
					if !util.ContainsInAnyInt32Array(shardID, this.config.Shards) {
						log.Debugf("shard %s-%s not enabled, skip processing", index, shardID)
						return errors.Errorf("shard %s-%v not enabled, skip processing", index, shardID)
					}
				}
			}

			shardInfo,err := metadata.GetPrimaryShardInfo(index, shardID)
			if err!=nil{
				return errors.Error("shardInfo was not found,", index, ",", shardID,",",err)
			}

			if shardInfo == nil {
				if rate.GetRateLimiter(fmt.Sprintf("shard_info_not_found_%v", index), util.IntToString(shardID), 1, 5, time.Minute*1).Allow() {
					log.Warn("shardInfo was not found,", index, ",", shardID)
				}
				return errors.Error("shardInfo was not found,", index, ",", shardID)
			}

			//write meta
			bufferKey = common.GetNodeLevelShuffleKey(clusterName, shardInfo.Node)

			if reshuffleType == "cluster"{
				bufferKey = common.GetClusterLevelShuffleKey(clusterName)
			} else if reshuffleType == "shard" {
				bufferKey = common.GetShardLevelShuffleKey(clusterName, index, shardID)
			}

			if global.Env().IsDebug {
				log.Tracef("%s/%s/%s => %v , %v", index, typeName, id, shardID, bufferKey)
			}

			////update actionItem
			buff, ok = docBuf[bufferKey]
			if !ok {

				var host string
				if reshuffleType=="cluster"{
					host =esConfig.ID
				}else{
					nodeInfo := metadata.GetNodeInfo(shardInfo.Node)
					if global.Env().IsDebug{
						log.Debugf("get node info by id:[%v], [%v]",shardInfo.Node,nodeInfo)
					}
					if nodeInfo == nil {
						if rate.GetRateLimiter("node_info_not_found_%v", shardInfo.Node, 1, 5, time.Minute*1).Allow() {
							log.Warnf("nodeInfo not found, %v %v", bufferKey, shardInfo.Node)
						}
						return errors.Errorf("nodeInfo not found, %v %v", bufferKey, shardInfo.Node)
					}
					host =nodeInfo.Http.PublishAddress
				}


				buff = bufferPool.Get()
				docBuf[bufferKey] = buff

				buffHosts[bufferKey] = host
				if global.Env().IsDebug {
					log.Debug(shardInfo.Index, ",", shardInfo.Shard, ",", host)
				}
			}

			//保存临时变量
			actionMeta.Write(metaBytes)

			return nil
		}, func(payloadBytes []byte) {

			if actionMeta.Len() > 0 {
				buff, ok := docBuf[bufferKey]
				if !ok {
					buff = bufferPool.Get()
					docBuf[bufferKey] = buff
				}
				if global.Env().IsDebug {
					log.Trace("metadata:", string(payloadBytes))
				}

				if buff.Len() > 0 {
					buff.Write(NEWLINEBYTES)
				}

				buff.Write(actionMeta.Bytes())
				if payloadBytes != nil && len(payloadBytes) > 0 {
					buff.Write(NEWLINEBYTES)
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

		submitMode := this.config.Mode //sync and async
		if submitMode == "sync" {

			for x, y := range docBuf {
				y.Write(NEWLINEBYTES)
				data := y.Bytes()

				if validateRequest {
					common.ValidateBulkRequest("sync-bulk", string(data))
				}

				endpoint, ok := buffHosts[x]
				if !ok {
					log.Error("shard endpoint was not found,", x)
					//TODO
					return
				}

				endpoint = path.Join(endpoint, pathStr)

				start := time.Now()
				code, status := this.bulkProcessor.Bulk(metadata, endpoint, data)
				stats.Timing("elasticsearch."+esConfig.Name+".bulk", "elapsed_ms", time.Since(start).Milliseconds())
				switch status {
				case SUCCESS:
					break
				case INVALID:
					log.Error("invalid bulk requests failed on endpoint,", x, ", code:", code)
					break
				case PARTIAL:
					log.Error("bulk requests partial failed on endpoint,", x, ", code:", code)
					break
				case FAILURE:
					log.Error("bulk requests failed on endpoint,", x, ", code:", code)
					return
				}

				ctx.SetDestination(fmt.Sprintf("%v:%v", "sync", x))
				bufferPool.Put(y)
			}

		} else {

			for x, y := range docBuf {
				y.Write(NEWLINEBYTES)
				data := y.Bytes()

				if validateRequest {
					common.ValidateBulkRequest("aync-bulk", string(data))
				}

				if len(data) > 0 {
					err := queue.Push(x, data)
					if err != nil {
						panic(err)
					}
					ctx.SetDestination(fmt.Sprintf("%v:%v", "async", x))
				} else {
					log.Warn("zero message,", x, ",", len(data), ",", string(body))
				}
				bufferPool.Put(y)
			}

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

		buffer := bytebufferpool.Get()

		buffer.Write(startPart)
		for i := 0; i < docCount; i++ {
			if i != 0 {
				buffer.Write([]byte(","))
			}
			buffer.Write(itemPart)
		}
		buffer.Write(endPart)

		ctx.Response.AppendBody(buffer.Bytes())
		bytebufferpool.Put(buffer)

		ctx.Response.SetStatusCode(200)
		fmt.Println("bulk reshuffle finished.")
		ctx.Finished()
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

var actionIndex = "index"
var actionDelete = "delete"
var actionCreate = "create"
var actionUpdate = "update"

var actionStart = []byte("\"")
var actionEnd = []byte("\"")

var actions = []string{"index", "delete", "create", "update"}

func parseActionMeta(data []byte) (action, index, typeName, id string) {

	match := false
	for _, v := range actions {
		jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			switch util.UnsafeBytesToString(key) {
			case "_index":
				index = string(value)
				break
			case "_type":
				typeName = string(value)
				break
			case "_id":
				id = string(value)
				break
			}
			match = true
			return nil
		}, v)
		action = v
		if match {
			return action, index, typeName, id
		}
	}

	log.Warn("fallback to unsafe parse:", util.UnsafeBytesToString(data))

	action = string(util.ExtractFieldFromBytes(&data, actionStart, actionEnd, nil))
	index, _ = jsonparser.GetString(data, action, "_index")
	typeName, _ = jsonparser.GetString(data, action, "_type")
	id, _ = jsonparser.GetString(data, action, "_id")

	if index != "" {
		return action, index, typeName, id
	}

	log.Warn("fallback to safety parse:", util.UnsafeBytesToString(data))
	return safetyParseActionMeta(data)
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

//performance is poor
func safetyParseActionMeta(scannedByte []byte) (action, index, typeName, id string) {

	////{ "index" : { "_index" : "test", "_id" : "1" } }
	var meta = elastic.BulkActionMetadata{}
	meta.UnmarshalJSON(scannedByte)
	if meta.Index != nil {
		index = meta.Index.Index
		typeName = meta.Index.Type
		id = meta.Index.ID
		action = actionIndex
	} else if meta.Create != nil {
		index = meta.Create.Index
		typeName = meta.Create.Type
		id = meta.Create.ID
		action = actionCreate
	} else if meta.Update != nil {
		index = meta.Update.Index
		typeName = meta.Update.Type
		id = meta.Update.ID
		action = actionUpdate
	} else if meta.Delete != nil {
		index = meta.Delete.Index
		typeName = meta.Delete.Type
		action = actionDelete
		id = meta.Delete.ID
	}

	return action, index, typeName, id
}
