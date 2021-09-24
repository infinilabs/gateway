package elastic

import (
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/elastic/model"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"path"
	"strconv"
	"sync"
	"time"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	param.Parameters
}

func (this BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}


func (this BulkReshuffle) Process(ctx *fasthttp.RequestCtx) {

	pathStr := util.UnsafeBytesToString(ctx.URI().Path())

	//拆解 bulk 请求，重新封装
	if util.SuffixStr(pathStr, "/_bulk") {

		ctx.Set(common.CACHEABLE, false)

		clusterName := this.MustGetString("elasticsearch")

		versionLock.RLock()
		esMajorVersion, ok := versions[clusterName]
		versionLock.RUnlock()

		if !ok {
			versionLock.Lock()
			esMajorVersion := elastic.GetClient(clusterName).GetMajorVersion()
			versions[clusterName] = esMajorVersion
			versionLock.Unlock()
		}

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
		buffEndpoints := map[string]string{}

		validEachLine := this.GetBool("validate_each_line", false)
		validMetadata := this.GetBool("validate_metadata", false)
		validPayload := this.GetBool("validate_payload", false)
		reshuffleType := this.GetStringOrDefault("level", "node")
		fixNullID := this.GetBool("fix_null_id", true) //sync and async

		enabledShards, checkShards := this.GetStringArray("shards")

		//renameMapping, resolveIndexRename := this.GetStringMap("index_rename")

		var buff *bytebufferpool.ByteBuffer

		var bufferKey string
		indexAnalysis := this.GetBool("index_stats_analysis", true)   //sync and async
		actionAnalysis := this.GetBool("action_stats_analysis", true) //sync and async
		validateRequest := this.GetBool("validate_request", false)
		actionMeta := smallSizedPool.Get()
		actionMeta.Reset()
		defer smallSizedPool.Put(actionMeta)

		var docBuffer []byte
		docBuffer=p.Get(this.GetIntOrDefault("doc_buffer_size",256*1024)) //doc buffer for bytes scanner
		defer p.Put(docBuffer)

		docCount, err := WalkBulkRequests(body,docBuffer, func(eachLine []byte) (skipNextLine bool) {
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

			//index_rename
			//if resolveIndexRename {
			//	for k, v := range renameMapping {
			//		if strings.Contains(k, "*") {
			//			patterns := radix.Compile(k) //TODO performance
			//			ok := patterns.Match(index)
			//			if ok {
			//				if global.Env().IsDebug {
			//					log.Debug("wildcard matched: ", pathStr)
			//				}
			//				index = v
			//				scannedByte = updateJsonWithNewIndex(scannedByte, index)
			//				break
			//			}
			//		} else if k == index {
			//			index = v
			//			scannedByte = updateJsonWithNewIndex(scannedByte, index)
			//			break
			//		}
			//	}
			//}

			var indexNew, typeNew, idNew string
			if index == "" && urlLevelIndex != "" {
				index = urlLevelIndex
				indexNew = urlLevelIndex
			}

			if typeName == "" && urlLevelType != "" {
				typeName = urlLevelType
				typeNew = urlLevelType
			}

			if (actionStr == actionIndex || actionStr == actionDelete) && len(id) == 0 && fixNullID {
				id = util.GetUUID()
				idNew = id
				if global.Env().IsDebug {
					log.Trace("generated ID,", id, ",", string(metaBytes))
				}
			}

			if indexNew != "" || typeNew != "" || idNew != "" {
				var err error
				metaBytes, err = updateJsonWithNewIndex(actionStr, metaBytes, indexNew, typeNew, idNew)
				if err != nil {
					panic(err)
				}
				if global.Env().IsDebug {
					log.Trace("updated meta,", id, ",", string(metaBytes))
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
				log.Warn("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", string(metaBytes))
				return errors.Error("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", string(metaBytes))
			}

			indexSettings, ok := metadata.Indices[index]

			if !ok {
				metadata = elastic.GetOrInitMetadata(esConfig)
				if global.Env().IsDebug {
					log.Trace("index was not found in index settings,", index, ",", string(metaBytes))
				}
				alias, ok := metadata.Aliases[index]
				if ok {
					if global.Env().IsDebug {
						log.Trace("found index in alias settings,", index, ",", string(metaBytes))
					}
					newIndex := alias.WriteIndex
					if alias.WriteIndex == "" {
						if len(alias.Index) == 1 {
							newIndex = alias.Index[0]
							if global.Env().IsDebug {
								log.Trace("found index in alias settings, no write_index, but only have one index, will use it,", index, ",", string(metaBytes))
							}
						} else {
							log.Warn("writer_index was not found in alias settings,", index, ",", alias)
							return errors.Error("writer_index was not found in alias settings,", index, ",", alias)
						}
					}
					indexSettings, ok = metadata.Indices[newIndex]
					if ok {
						if global.Env().IsDebug {
							log.Trace("index was found in index settings,", index, "=>", newIndex, ",", metaStr, ",", indexSettings)
						}
						index = newIndex
						goto CONTINUE_RESHUFFLE
					} else {
						if global.Env().IsDebug {
							log.Trace("writer_index was not found in index settings,", index, ",", string(metaBytes))
						}
					}
				} else {
					if global.Env().IsDebug {
						log.Trace("index was not found in alias settings,", index, ",", string(metaBytes))
					}
				}

				if rate.GetRateLimiter("index_setting_not_found", index, 1, 5, time.Minute*1).Allow() {
					log.Warn("index setting not found,", index, ",", string(metaBytes))
				}

				return errors.Error("index setting not found,", index, ",", string(metaBytes))
			}

		CONTINUE_RESHUFFLE:

			if indexSettings.Shards <= 0 || indexSettings.Status == "close" {
				log.Debugf("index %v closed,", indexSettings.Index)
				return errors.Errorf("index %v closed,", indexSettings.Index)
			}

			if indexSettings.Shards != 1 {
				//如果 shards=1，则直接找主分片所在节点，否则计算一下。
				shardID = elastic.GetShardID(esMajorVersion, []byte(id), indexSettings.Shards)

				if global.Env().IsDebug {
					log.Tracef("%s/%s => %v", index, id, shardID)
				}

				//save endpoint for bufferkey
				if checkShards && len(enabledShards) > 0 {
					if !util.ContainsAnyInArray(strconv.Itoa(shardID), enabledShards) {
						log.Debugf("shard %s-%s not enabled, skip processing", index, shardID)
						//skipNext = true
						//continue
						return errors.Errorf("shard %s-%v not enabled, skip processing", index, shardID)
					}
				}

			}

			shardInfo := metadata.GetPrimaryShardInfo(index, shardID)
			if shardInfo == nil {
				if rate.GetRateLimiter(fmt.Sprintf("shard_info_not_found_%v", index), util.IntToString(shardID), 1, 5, time.Minute*1).Allow() {
					log.Warn("shardInfo was not found,", index, ",", shardID)
				}
				return errors.Error("shardInfo was not found,", index, ",", shardID)
			}

			//write meta
			bufferKey = common.GetNodeLevelShuffleKey(clusterName, shardInfo.NodeID)
			if reshuffleType == "shard" {
				bufferKey = common.GetShardLevelShuffleKey(clusterName, index, shardID)
			}

			if global.Env().IsDebug {
				log.Tracef("%s/%s/%s => %v , %v", index, typeName, id, shardID, bufferKey)
			}

			////update actionItem
			buff, ok = docBuf[bufferKey]
			if !ok {
				nodeInfo := metadata.GetNodeInfo(shardInfo.NodeID)
				if nodeInfo == nil {
					if rate.GetRateLimiter("node_info_not_found_%v", shardInfo.NodeID, 1, 5, time.Minute*1).Allow() {
						log.Warnf("nodeInfo not found, %v %v", bufferKey, shardInfo.NodeID)
					}
					return errors.Errorf("nodeInfo not found, %v %v", bufferKey, shardInfo.NodeID)
				}

				buff = bufferPool.Get()
				buff.Reset()
				docBuf[bufferKey] = buff

				buffEndpoints[bufferKey] = nodeInfo.Http.PublishAddress
				if global.Env().IsDebug {
					log.Debug(shardInfo.Index, ",", shardInfo.ShardID, ",", nodeInfo.Http.PublishAddress)
				}
			}

			//保存临时变量
			actionMeta.Write(metaBytes)

			return nil
		}, func(payloadBytes []byte) {

			if actionMeta.Len() > 0 {
				buff, ok = docBuf[bufferKey]
				if !ok {
					buff = bufferPool.Get()
					buff.Reset()
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
			if global.Env().IsDebug{
				log.Error(err)
			}
			return
		}

		submitMode := this.GetStringOrDefault("mode", "sync") //sync and async
		if submitMode == "sync" {
			bulkProcessor := BulkProcessor{
				RotateConfig: rotate.RotateConfig{
					Compress:     this.GetBool("compress_after_rotate", true),
					MaxFileAge:   this.GetIntOrDefault("max_file_age", 0),
					MaxFileCount: this.GetIntOrDefault("max_file_count", 100),
					MaxFileSize:  this.GetIntOrDefault("max_file_size_in_mb", 1024),
				},
				Config: BulkProcessorConfig{
					Compress:                  this.GetBool("compress", false),
					LogInvalidMessage:         this.GetBool("log_invalid_message", true),
					LogInvalid200Message:      this.GetBool("log_invalid_200_message", true),
					LogInvalid200RetryMessage: this.GetBool("log_200_retry_message", true),
					Log429RetryMessage:        this.GetBool("log_429_retry_message", true),
					RetryDelayInSeconds:       this.GetIntOrDefault("retry_delay_in_seconds", 1),
					RejectDelayInSeconds:      this.GetIntOrDefault("reject_retry_delay_in_seconds", 1),
					MaxRejectRetryTimes:       this.GetIntOrDefault("max_reject_retry_times", 3),
					MaxRetryTimes:             this.GetIntOrDefault("max_retry_times", 3),
					MaxRequestBodySize:        this.GetIntOrDefault("max_logged_request_body_size", 1024),
					MaxResponseBodySize:       this.GetIntOrDefault("max_logged_response_body_size", 1024),

					SaveFailure:       this.GetBool("save_failure",true),
					FailureRequestsQueue:       this.GetStringOrDefault("failure_queue",fmt.Sprintf("%v-failure",clusterName)),
					InvalidRequestsQueue:       this.GetStringOrDefault("invalid_queue",fmt.Sprintf("%v-invalid",clusterName)),
					DeadRequestsQueue:       	this.GetStringOrDefault("dead_queue",fmt.Sprintf("%v-dead",clusterName)),
					DocBufferSize: this.GetIntOrDefault("doc_buffer_size",256*1024),
				},
			}

			for x, y := range docBuf {
				y.Write(NEWLINEBYTES)
				data := y.Bytes()

				if validateRequest {
					common.ValidateBulkRequest("sync-bulk", string(data))
				}

				endpoint, ok := buffEndpoints[x]
				if !ok {
					log.Error("shard endpoint was not found,", x)
					//TODO
					return
				}

				endpoint = path.Join(endpoint, pathStr)

				start:=time.Now()
				code, status := bulkProcessor.Bulk(metadata, endpoint, data, fastHttpClient)
				stats.Timing("elasticsearch."+esConfig.Name+".bulk","elapsed_ms",time.Since(start).Milliseconds())
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


var actionIndex ="index"
var actionDelete = "delete"
var actionCreate = "create"
var actionUpdate = "update"

var actionStart = []byte("\"")
var actionEnd = []byte("\"")

var actions = []string{"index","delete","create","update"}

func parseActionMeta(data []byte) (action, index, typeName, id string) {

	match:=false
	for _,v:=range actions{
		jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			switch util.UnsafeBytesToString(key) {
			case "_index":
				index=string(value)
				break
			case "_type":
				typeName=string(value)
				break
			case "_id":
				id=string(value)
				break
			}
			match=true
			return nil
		}, v)
		action=v
		if match{
			//fmt.Println(action,",",index,",",typeName,",", id)
			return action, index,typeName, id
		}
	}

	log.Warn("fallback to unsafe parse:",util.UnsafeBytesToString(data))

	action = string(util.ExtractFieldFromBytes(&data, actionStart, actionEnd, nil))
	index,_=jsonparser.GetString(data,action,"_index")
	typeName,_=jsonparser.GetString(data,action,"_type")
	id,_=jsonparser.GetString(data,action,"_id")

	if index!=""{
		return action, index,typeName, id
	}

	log.Warn("fallback to safety parse:",util.UnsafeBytesToString(data))
	return safetyParseActionMeta(data)
}

func updateJsonWithNewIndex(action string,scannedByte []byte, index, typeName, id string) (newBytes []byte,err error) {

	if global.Env().IsDebug{
		log.Trace("update:",action,",",index,",",typeName,",",id)
	}

	newBytes= make([]byte,len(scannedByte))
	copy(newBytes,scannedByte)

	if index != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+index+"\""),action,"_index")
		if err!=nil{
			return newBytes,err
		}
	}
	if typeName != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+typeName+"\""),action,"_type")
		if err!=nil{
			return newBytes,err
		}
	}
	if id != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+id+"\""),action,"_id")
		if err!=nil{
			return newBytes,err
		}
	}

	return newBytes,err
}

//performance is poor
func safetyParseActionMeta(scannedByte []byte) (action , index, typeName, id string) {

	////{ "index" : { "_index" : "test", "_id" : "1" } }
	var meta = model.BulkActionMetadata{}
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

var versions = map[string]int{}
var versionLock = sync.RWMutex{}
