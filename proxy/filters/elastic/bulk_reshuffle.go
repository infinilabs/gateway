package elastic

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/valyala/bytebufferpool"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	param.Parameters
}

func (this BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}

var bufferPool bytebufferpool.Pool

var actionIndex = []byte("index")
var actionDelete = []byte("delete")
var actionCreate = []byte("create")
var actionUpdate = []byte("update")

var actionStart = []byte("\"")
var actionEnd = []byte("\"")

var indexStart = []byte("\"_index\"")
var indexEnd = []byte("\"")

var filteredFromValue = []byte(": \"")

var idStart = []byte("\"_id\"")
var idEnd = []byte("\"")

func parseActionMeta(data []byte) ([]byte, []byte, []byte) {

	action := util.ExtractFieldFromBytes(&data, actionStart, actionEnd, nil)
	index := util.ExtractFieldFromBytesWitSkipBytes(&data, indexStart, []byte("\""), indexEnd, filteredFromValue)
	id := util.ExtractFieldFromBytesWitSkipBytes(&data, idStart, []byte("\""), idEnd, filteredFromValue)

	return action, index, id
}

//"_index":"test" => "_index":"test", "_id":"id"
func insertUUID(scannedByte []byte) (newBytes []byte, id string) {
	id = util.GetUUID()
	newData := util.InsertBytesAfterField(&scannedByte, []byte("\"_index\""), []byte("\""), []byte("\""), []byte(",\"_id\":\""+id+"\""))
	return newData, id
}

//TODO performance
func updateJsonWithUUID(scannedByte []byte) (newBytes []byte, id string) {
	var meta elastic.BulkActionMetadata
	meta = elastic.BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte, &meta)
	id = util.GetUUID()
	if meta.Index != nil {
		meta.Index.ID = id
	} else if meta.Create != nil {
		meta.Create.ID = id
	}
	return util.MustToJSONBytes(meta), id
}

//TODO performance
func updateJsonWithNewIndex(scannedByte []byte, index string) (newBytes []byte) {
	var meta elastic.BulkActionMetadata
	meta = elastic.BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte, &meta)
	if meta.Index != nil {
		meta.Index.Index = index
	} else if meta.Create != nil {
		meta.Create.Index = index
	} else if meta.Update != nil {
		meta.Update.Index = index
	} else if meta.Delete != nil {
		meta.Delete.Index = index
	}
	return util.MustToJSONBytes(meta)
}

func parseJson(scannedByte []byte) (action []byte, index, id string) {
	//use Json
	var meta = elastic.BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte, &meta)

	if meta.Index != nil {
		index = meta.Index.Index
		id = meta.Index.ID
		action = actionIndex
	} else if meta.Create != nil {
		index = meta.Create.Index
		id = meta.Create.ID
		action = actionCreate
	} else if meta.Update != nil {
		index = meta.Update.Index
		id = meta.Update.ID
		action = actionUpdate
	} else if meta.Delete != nil {
		index = meta.Delete.Index
		action = actionDelete
		id = meta.Delete.ID
	}

	return action, index, id
}

var versions = map[string]int{}
var versionLock = sync.Mutex{}

func (this BulkReshuffle) Process(ctx *fasthttp.RequestCtx) {

	path := string(ctx.URI().Path())

	//TODO 处理 {INDEX}/_bulk 的情况
	//filebeat 等都是 bulk 结尾的请求了。
	//需要拆解 bulk 请求，重新封装
	if util.PrefixStr(path, "/_bulk") {

		ctx.Set(common.CACHEABLE, false)

		clusterName := this.MustGetString("elasticsearch")
		esMajorVersion, ok := versions[clusterName]
		if !ok {
			versionLock.Lock()
			esMajorVersion := elastic.GetClient(clusterName).GetMajorVersion()
			versions[clusterName] = esMajorVersion
			versionLock.Unlock()
		}

		metadata := elastic.GetMetadata(clusterName)
		if metadata == nil {
			if rate.GetRateLimiter("cluster_metadata", clusterName, 1, 1, 5*time.Second).Allow() {
				log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle", clusterName)
			}
			time.Sleep(10 * time.Second)
			return
		}

		esConfig := elastic.GetConfig(clusterName)

		safetyParse := this.GetBool("safety_parse", true)
		validMetadata := this.GetBool("valid_metadata", false)
		validateRequest := this.GetBool("validate_request", false)
		reshuffleType := this.GetStringOrDefault("level", "node")
		submitMode := this.GetStringOrDefault("mode", "sync")         //sync and async
		fixNullID := this.GetBool("fix_null_id", true)                //sync and async
		indexAnalysis := this.GetBool("index_stats_analysis", true)   //sync and async
		actionAnalysis := this.GetBool("action_stats_analysis", true) //sync and async
		enabledShards, checkShards := this.GetStringArray("shards")

		renameMapping, resolveIndexRename := this.GetStringMap("index_rename")

		body := ctx.Request.GetRawBody()
		if validateRequest {
			common.ValidateBulkRequest("raw_body", string(body))
		}

		scanner := bufio.NewScanner(bytes.NewReader(body))
		scanner.Split(util.GetSplitFunc([]byte("\n")))
		nextIsMeta := true

		//index-shardID -> buffer
		docBuf := map[string]*bytebufferpool.ByteBuffer{}
		buffEndpoints := map[string]string{}
		skipNext := false
		var buff *bytebufferpool.ByteBuffer
		var indexStatsData map[string]int
		var actionStatsData map[string]int
		var indexStatsLock sync.Mutex

		actionMeta := bufferPool.Get()
		defer bufferPool.Put(actionMeta)

		var needActionBody = true
		var bufferKey string
		var docCount = 0

		for scanner.Scan() {
			shardID := 0
			scannedByte := scanner.Bytes()
			if scannedByte == nil || len(scannedByte) <= 0 {
				log.Debug("invalid scanned byte, continue")
				continue
			}

			if validateRequest {
				common.ValidateBulkRequest("scanned_byte", string(scannedByte))
			}

			if skipNext {
				skipNext = false
				nextIsMeta = true
				log.Debug("skip body processing")
				continue
			}

			if nextIsMeta {
				nextIsMeta = false

				var index string
				var id string
				var action []byte

				if safetyParse {
					action, index, id = parseJson(scannedByte)
				} else {
					var indexb, idb []byte

					//TODO action: update ,index:  ,id: 1,_indextest
					//{ "update" : {"_id" : "1", "_index" : "test"} }
					//字段顺序换了。
					action, indexb, idb = parseActionMeta(scannedByte)
					index = string(indexb)
					id = string(idb)

					if len(action) == 0 || index == "" {
						log.Warn("invalid bulk action:", string(action), ",index:", string(indexb), ",id:", string(idb), ", try json parse:", string(scannedByte))
						action, index, id = parseJson(scannedByte)
					}
				}

				//index_rename
				if resolveIndexRename {
					for k, v := range renameMapping {
						if strings.Contains(k, "*") {
							patterns := radix.Compile(k) //TODO performance
							ok := patterns.Match(index)
							if ok {
								if global.Env().IsDebug {
									log.Debug("wildcard matched: ", path)
								}
								index = v
								scannedByte = updateJsonWithNewIndex(scannedByte, index)
								break
							}
						} else if k == index {
							index = v
							scannedByte = updateJsonWithNewIndex(scannedByte, index)
							break
						}
					}
				}

				//统计索引次数
				stats.Increment("elasticsearch."+clusterName+".indexing", index)
				if indexAnalysis {
					//init
					if indexStatsData == nil {
						indexStatsLock.Lock()
						if indexStatsData == nil {
							indexStatsData = map[string]int{}
						}
						indexStatsLock.Unlock()
					}

					//stats
					indexStatsLock.Lock()
					v, ok := indexStatsData[index]
					if !ok {
						indexStatsData[index] = 1
					} else {
						indexStatsData[index] = v + 1
					}
					indexStatsLock.Unlock()
				}

				if actionAnalysis {
					//init
					if actionStatsData == nil {
						indexStatsLock.Lock()
						if actionStatsData == nil {
							actionStatsData = map[string]int{}
						}
						indexStatsLock.Unlock()
					}

					//stats
					indexStatsLock.Lock()
					actionStr := string(action)
					v, ok := actionStatsData[actionStr]
					if !ok {
						actionStatsData[actionStr] = 1
					} else {
						actionStatsData[actionStr] = v + 1
					}
					indexStatsLock.Unlock()
				}

				if (bytes.Equal(action, []byte("index")) || bytes.Equal(action, []byte("create"))) && len(id) == 0 && fixNullID {
					if safetyParse {
						scannedByte, id = updateJsonWithUUID(scannedByte)
					} else {
						scannedByte, id = insertUUID(scannedByte)
					}
					if global.Env().IsDebug {
						log.Trace("generated ID,", id, ",", string(scannedByte))
					}
				}

				if validMetadata {
					obj := map[string]interface{}{}
					err := util.FromJSONBytes(scannedByte, &obj)
					if err != nil {
						log.Error("error on validate action metadata")
						panic(err)
					}
				}

				if len(action) == 0 || index == "" || id == "" {
					log.Warn("invalid bulk action:", string(action), ",index:", string(index), ",id:", string(id), ",", string(scannedByte))
					return
				}

				indexSettings, ok := metadata.Indices[index]

				if !ok {
					metadata = elastic.GetMetadata(clusterName)
					if global.Env().IsDebug {
						log.Trace("index was not found in index settings,", index, ",", string(scannedByte))
					}
					alias, ok := metadata.Aliases[index]
					if ok {
						if global.Env().IsDebug {
							log.Trace("found index in alias settings,", index, ",", string(scannedByte))
						}
						newIndex := alias.WriteIndex
						if alias.WriteIndex == "" {
							if len(alias.Index) == 1 {
								newIndex = alias.Index[0]
								if global.Env().IsDebug {
									log.Trace("found index in alias settings, no write_index, but only have one index, will use it,", index, ",", string(scannedByte))
								}
							} else {
								log.Warn("writer_index was not found in alias settings,", index, ",", alias)
								return
							}
						}
						indexSettings, ok = metadata.Indices[newIndex]
						if ok {
							if global.Env().IsDebug {
								log.Trace("index was found in index settings,", index, "=>", newIndex, ",", string(scannedByte), ",", indexSettings)
							}
							index = newIndex
							goto CONTINUE_RESHUFFLE
						} else {
							if global.Env().IsDebug {
								log.Trace("writer_index was not found in index settings,", index, ",", string(scannedByte))
							}
						}
					} else {
						if global.Env().IsDebug {
							log.Trace("index was not found in alias settings,", index, ",", string(scannedByte))
						}
					}

					if rate.GetRateLimiter("index_setting_not_found", index, 1, 5, time.Minute*1).Allow() {
						log.Warn("index setting not found,", index, ",", string(scannedByte))
					}

					return
				}

			CONTINUE_RESHUFFLE:

				if indexSettings.Shards != 1 {
					//如果 shards=1，则直接找主分片所在节点，否则计算一下。
					shardID = elastic.GetShardID(esMajorVersion, []byte(id), indexSettings.Shards)

					if global.Env().IsDebug {
						log.Tracef("%s/%s => %v", index, id, shardID)
					}

					//save endpoint for bufferkey
					if checkShards && len(enabledShards) > 0 && needActionBody {
						if !util.ContainsAnyInArray(strconv.Itoa(shardID), enabledShards) {
							log.Debugf("shard %s-%s not enabled, skip processing", index, shardID)
							skipNext = true
							continue
						}
					}

				}

				shardInfo := metadata.GetPrimaryShardInfo(index, shardID)
				if shardInfo == nil {
					if rate.GetRateLimiter(fmt.Sprintf("shard_info_not_found_%v", index), util.IntToString(shardID), 1, 5, time.Minute*1).Allow() {
						log.Warn("shardInfo was not found,", index, ",", shardID)
					}
					return
				}

				//write meta
				bufferKey = common.GetNodeLevelShuffleKey(clusterName, shardInfo.NodeID)
				if reshuffleType == "shard" {
					bufferKey = common.GetShardLevelShuffleKey(clusterName, index, shardID)
				}

				if global.Env().IsDebug {
					log.Tracef("%s/%s => %v , %v", index, id, shardID, bufferKey)
				}

				////update actionItem
				buff, ok = docBuf[bufferKey]
				if !ok {
					nodeInfo := metadata.GetNodeInfo(shardInfo.NodeID)
					if nodeInfo == nil {
						if rate.GetRateLimiter("node_info_not_found_%v", shardInfo.NodeID, 1, 5, time.Minute*1).Allow() {
							log.Warn("nodeInfo not found,", shardID, ",", shardInfo.NodeID)
						}
						return
					}

					buff = bufferPool.Get()
					docBuf[bufferKey] = buff

					buffEndpoints[bufferKey] = nodeInfo.Http.PublishAddress
					if global.Env().IsDebug {
						log.Debug(shardInfo.Index, ",", shardInfo.ShardID, ",", nodeInfo.Http.PublishAddress)
					}
				}

				//保存临时变量
				actionMeta.Write(scannedByte)

				if bytes.Equal(action, actionDelete) {
					nextIsMeta = true
					needActionBody = false
				}
				docCount++
			} else {
				nextIsMeta = true

				if actionMeta.Len() > 0 {
					buff, ok = docBuf[bufferKey]
					if !ok {
						buff = bufferPool.Get()
						docBuf[bufferKey] = buff
					}
					if global.Env().IsDebug {
						log.Trace("metadata:", string(scannedByte))
					}
					if validateRequest {
						common.ValidateBulkRequest("before_write_meta", string(buff.String()))
						common.ValidateBulkRequest("validate_meta", string(actionMeta.String()))
					}

					buff.Write(actionMeta.Bytes())
					actionMeta.Reset()

					if validateRequest {
						common.ValidateBulkRequest("after_write_meta", string(buff.String()))
					}

					buff.WriteString("\n")
					if needActionBody && scannedByte != nil && len(scannedByte) > 0 {
						if validateRequest {
							common.ValidateBulkRequest("before_write_body", string(buff.String()))
						}
						buff.Write(scannedByte)

						if validateRequest {
							common.ValidateBulkRequest("after_write_body", string(buff.String()))
						}

						buff.WriteString("\n")
					}

					//clearup actionMeta
					actionMeta.Reset()
				}

			}
		}

		log.Debugf("total [%v] operations in bulk requests", docCount)

		for x, y := range docBuf {
			if submitMode == "sync" {
				endpoint, ok := buffEndpoints[x]
				if !ok {
					log.Error("shard endpoint was not found,", x)
					//TODO
					return
				}

				data := y.Bytes()
				if validateRequest {
					common.ValidateBulkRequest("sync-bulk", string(data))
				}

				status, ok := this.Bulk(esConfig, endpoint, data)
				if !ok {
					log.Error("bulk failed on endpoint,", x, ", code:", status)
					//TODO
					return
				}

				ctx.SetDestination(fmt.Sprintf("%v:%v", "sync", x))
			} else {
				data := y.Bytes()
				if validateRequest {
					common.ValidateBulkRequest("initial-enqueue", string(data))
				}
				if len(data) > 0 {
					err := queue.Push(x, data)
					if err != nil {
						panic(err)
					}
					ctx.SetDestination(fmt.Sprintf("%v:%v", "async", x))
				} else {
					log.Warn("zero message,", x, ",", y.Len(), ",", string(body))
				}
			}
			y.Reset()
			bufferPool.Put(y)
		}

		if indexAnalysis {
			ctx.Set("bulk_index_stats", indexStatsData)
		}
		if actionAnalysis {
			ctx.Set("bulk_action_stats", actionStatsData)
		}

		if ctx.Has("elastic_cluster_name") {
			es1 := ctx.MustGetStringArray("elastic_cluster_name")
			ctx.Set("elastic_cluster_name", append(es1, clusterName))
		} else {
			ctx.Set("elastic_cluster_name", []string{clusterName})
		}

		//fake results

		ctx.SetContentType(JSON_CONTENT_TYPE)
		ctx.WriteString("{\"took\":0,\"errors\":false,\"items\":[")
		for i := 0; i < docCount; i++ {
			if i != 0 {
				ctx.WriteString(",")
			}
			ctx.WriteString("{\"index\":{\"_index\":\"fake-index\",\"_type\":\"doc\",\"_id\":\"1\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":1,\"successful\":1,\"failed\":0},\"_seq_no\":1,\"_primary_term\":1,\"status\":200}}")
		}
		ctx.WriteString("]}")
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
		return
	}

	return

	//处理单次请求。
	pathItems := strings.Split(path, "/")
	if len(pathItems) != 4 {
		//fmt.Println("not a valid indexing request,",len(pathItems),pathItems)
		return
	}

	return
	//排除条件，非 _ 开头的索引。
	//可以指定排除和允许的索引，设置匹配的索引名称，通配符。

	//PUT/POST index/_doc/UUID
	//只有匹配到是单独的索引请求才会进行合并处理。
	//放内存里面，按节点或者分片为单位进行缓存，或者固定的通道数，固定通道数<按节点<按分片。
	//count、size 和 timeout 任意满足即进行 bulk 提交。

	//通过 ID 获取到分片所在节点位置，没有 ID 就获取到包含主分片的节点，均衡选择，或者主动生成 ID。

	//变成 bulk 格式

	//defer writerPool.Put(w)
	//
	//err := request.MarshalFastJSON(w)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = queue.Push(this.GetStringOrDefault("queue_name","request_logging"),w.Bytes() )
	//if err != nil {
	//	panic(err)
	//}

}

//TODO 提取出来，作为公共方法，和 indexing/bulking_indexing 的方法合并

var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func (joint BulkReshuffle) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data []byte) (int, bool) {
	if data == nil || len(data) == 0 {
		log.Error("data size is empty,", endpoint)
		return 0, true
	}

	if cfg.IsTLS() {
		endpoint = "https://" + endpoint
	} else {
		endpoint = "http://" + endpoint
	}
	url := fmt.Sprintf("%s/_bulk", endpoint)
	compress := joint.GetBool("compress", true)

	req := fasthttp.AcquireRequest()
	req.Reset()
	req.ResetBody()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("bulk_indexing")

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	req.Header.SetContentType("application/json")

	if cfg.BasicAuth != nil {
		req.URI().SetUsername(cfg.BasicAuth.Username)
		req.URI().SetPassword(cfg.BasicAuth.Password)
	}

	if len(data) > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			//req.SetBody(body)
			req.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.Write(data)
				w.Flush()
			})

		}
	}
	retryTimes := 0

DO:

	err := fastHttpClient.Do(req, resp)
	if resp == nil {
		if global.Env().IsDebug {
			log.Error(err)
		}
		return 0, false
	}

	if err != nil {
		if global.Env().IsDebug {
			log.Error(err)
		}
		return resp.StatusCode(), false
	}

	// Do we need to decompress the response?
	var resbody = resp.GetRawBody()
	if global.Env().IsDebug {
		log.Trace(resp.StatusCode(), string(resbody))
	}

	if resp.StatusCode() == 400 {

		if joint.GetBool("log_400_message", true) {

			if rate.GetRateLimiter("log_400_message", endpoint, 1, 1, 5*time.Second).Allow() {
				log.Warnf("elasticsearch [%v] code 400", endpoint)
			}

			path1 := path.Join(global.Env().GetWorkingDir(), "bulk_400_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("URL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody := data
			resBody1 := resbody
			if truncateSize > 0 {
				if len(reqBody) > truncateSize {
					reqBody = reqBody[:truncateSize]
				}
				if len(resBody1) > truncateSize {
					resBody1 = resBody1[:truncateSize]
				}
			}
			util.FileAppendNewLineWithByte(path1, reqBody)
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)
		}
		return resp.StatusCode(), false
	}

	//TODO check respbody's error
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		//200{"took":2,"errors":true,"items":[
		if resp.StatusCode() == http.StatusOK {
			//handle error items
			//"errors":true
			hit := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
			if hit {
				if joint.GetBool("log_bulk_message", false) {
					path1 := path.Join(global.Env().GetWorkingDir(), "bulk_req_failure.log")
					truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
					util.FileAppendNewLineWithByte(path1, []byte("URL:"))
					util.FileAppendNewLineWithByte(path1, []byte(url))
					util.FileAppendNewLineWithByte(path1, []byte("Request:"))
					reqBody := data
					resBody1 := resbody
					if truncateSize > 0 {
						if len(reqBody) > truncateSize {
							reqBody = reqBody[:truncateSize]
						}
						if len(resBody1) > truncateSize {
							resBody1 = resBody1[:truncateSize]
						}
					}
					util.FileAppendNewLineWithByte(path1, reqBody)
					util.FileAppendNewLineWithByte(path1, []byte("Response:"))
					util.FileAppendNewLineWithByte(path1, resBody1)
				}
				if joint.GetBool("warm_retry_message", false) {
					log.Warnf("elasticsearch bulk error, retried %v times, will try again", retryTimes)
				}

				retryTimes++
				delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
				time.Sleep(time.Duration(delayTime) * time.Second)
				goto DO
			}
		}

		return resp.StatusCode(), true
	} else if resp.StatusCode() == 429 {
		log.Warnf("elasticsearch rejected, retried %v times, will try again", retryTimes)
		delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
		time.Sleep(time.Duration(delayTime) * time.Second)
		if retryTimes > 300 {
			if joint.GetBool("warm_retry_message", false) {
				log.Errorf("elasticsearch rejected, retried %v times, quit retry", retryTimes)
			}
			return resp.StatusCode(), false
		}
		retryTimes++
		goto DO
	} else {
		if joint.GetBool("log_bulk_message", true) {
			path1 := path.Join(global.Env().GetWorkingDir(), "bulk_error_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("URL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody := data
			resBody1 := resbody
			if truncateSize > 0 {
				if len(reqBody) > truncateSize {
					reqBody = reqBody[:truncateSize-1]
				}
				if len(resBody1) > truncateSize {
					resBody1 = resBody1[:truncateSize-1]
				}
			}
			util.FileAppendNewLineWithByte(path1, reqBody)
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)

		}
		if joint.GetBool("warm_retry_message", false) {
			log.Errorf("invalid bulk response, %v - %v", resp.StatusCode(), string(resbody))
		}
		return resp.StatusCode(), false
	}
	return resp.StatusCode(), true
}
