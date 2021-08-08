package elastic

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var bufferPool =bytebufferpool.NewPool(65536,655360)
var smallSizedPool =bytebufferpool.NewPool(512,655360)

var  NEWLINEBYTES =[]byte("\n")

func (this BulkReshuffle) Process(ctx *fasthttp.RequestCtx) {

	pathStr := string(ctx.URI().Path())

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

		//renameMapping, resolveIndexRename := this.GetStringMap("index_rename")

		body := ctx.Request.GetRawBody()

		scanner := bufio.NewScanner(bytes.NewReader(body))
		scanner.Split(util.GetSplitFunc(NEWLINEBYTES))
		nextIsMeta := true

		//index-shardID -> buffer
		docBuf := map[string]*bytebufferpool.ByteBuffer{}
		buffEndpoints := map[string]string{}
		skipNext := false
		var buff *bytebufferpool.ByteBuffer
		var indexStatsData map[string]int
		var actionStatsData map[string]int
		var indexStatsLock sync.Mutex

		actionMeta := smallSizedPool.Get()
		actionMeta.Reset()
		defer smallSizedPool.Put(actionMeta)

		var needActionBody = true
		var bufferKey string
		var docCount = 0

		for scanner.Scan() {
			shardID := 0
			scannedByte := scanner.Bytes()
			scannedStr:=string(scannedByte)
			if scannedByte == nil || len(scannedByte) <= 0 {
				log.Debug("invalid scanned byte, continue")
				continue
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
				var typeName string
				pathArray := strings.Split(pathStr, "/")

				var urlLevelIndex string
				var urlLevelType string
				switch len(pathArray) {
				case 4:
					urlLevelIndex = pathArray[1]
					urlLevelType = pathArray[2]
					break
				case 3:
					urlLevelIndex = pathArray[1]
					break
				}

				var id string
				var action []byte

				if safetyParse {
					action, index, typeName, id = parseJson(scannedByte)
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
						action, index, typeName, id = parseJson(scannedByte)
					}
				}

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

				if (bytes.Equal(action, []byte("index")) || bytes.Equal(action, []byte("create"))) && len(id) == 0 && fixNullID {
					id = util.GetUUID()
					idNew = id
					if global.Env().IsDebug {
						log.Trace("generated ID,", id, ",", string(scannedByte))
					}
				}

				if indexNew != "" || typeNew != "" || idNew != "" {
					scannedByte = updateJsonWithNewIndex(scannedByte, indexNew, typeNew, idNew)
					if global.Env().IsDebug {
						log.Trace("updated meta,", id, ",", string(scannedByte))
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
								log.Trace("index was found in index settings,", index, "=>", newIndex, ",", scannedStr, ",", indexSettings)
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

				if indexSettings.Shards<=0 ||indexSettings.Status=="close"{
					log.Debugf("index %v closed,",indexSettings.Index)
					return
				}

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
					log.Tracef("%s/%s/%s => %v , %v", index, typeName, id, shardID, bufferKey)
				}

				////update actionItem
				buff, ok = docBuf[bufferKey]
				if !ok {
					nodeInfo := metadata.GetNodeInfo(shardInfo.NodeID)
					if nodeInfo == nil {
						if rate.GetRateLimiter("node_info_not_found_%v", shardInfo.NodeID, 1, 5, time.Minute*1).Allow() {
							log.Warnf("nodeInfo not found, %v %v", bufferKey,shardInfo.NodeID)
						}
						return
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
						buff.Reset()
						docBuf[bufferKey] = buff
					}
					if global.Env().IsDebug {
						log.Trace("metadata:", string(scannedByte))
					}

					if buff.Len() > 0 {
						buff.Write(NEWLINEBYTES)
					}

					buff.Write(actionMeta.Bytes())
					actionMeta.Reset()

					if needActionBody && scannedByte != nil && len(scannedByte) > 0 {
						buff.Write(NEWLINEBYTES)
						buff.Write(scannedByte)
					}

					//cleanup actionMeta
					actionMeta.Reset()
				}

			}
		}

		log.Tracef("total [%v] operations in bulk requests", docCount)

		for x, y := range docBuf {
			y.Write(NEWLINEBYTES)
			data := y.Bytes()

			if validateRequest {
				common.ValidateBulkRequest("sync-bulk", string(data))
			}

			if submitMode == "sync" {
				endpoint, ok := buffEndpoints[x]
				if !ok {
					log.Error("shard endpoint was not found,", x)
					//TODO
					return
				}

				endpoint = path.Join(endpoint, pathStr)

				status, ok := bulkProcessor.Bulk(esConfig, endpoint, data,fastHttpClient)
				if !ok {
					log.Error("bulk failed on endpoint,", x, ", code:", status)
					//TODO
					return
				}

				ctx.SetDestination(fmt.Sprintf("%v:%v", "sync", x))
			} else {
				if len(data) > 0 {
					err := queue.Push(x, data)
					if err != nil {
						panic(err)
					}
					ctx.SetDestination(fmt.Sprintf("%v:%v", "async", x))
				} else {
					log.Warn("zero message,", x, ",", len(data), ",", string(body))
				}
			}

			//fmt.Println("length of buff:",y.Len())
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
		ctx.Write(startPart)
		for i := 0; i < docCount; i++ {
			if i != 0 {
				ctx.Write([]byte(","))
			}
			ctx.Write(itemPart)
		}
		ctx.Write(endPart)
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
var startPart=[]byte("{\"took\":0,\"errors\":false,\"items\":[")
var itemPart=[]byte("{\"index\":{\"_index\":\"fake-index\",\"_type\":\"doc\",\"_id\":\"1\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":1,\"successful\":1,\"failed\":0},\"_seq_no\":1,\"_primary_term\":1,\"status\":200}}")
var endPart=[]byte("]}")

//TODO 提取出来，作为公共方法，和 indexing/bulking_indexing 的方法合并
var fastHttpClient = &fasthttp.Client{
	MaxConnDuration:     0,
	MaxIdleConnDuration: 0,
	ReadTimeout:         time.Second * 60,
	WriteTimeout:        time.Second * 60,
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

var bulkProcessor=BulkProcessor{
	RotateConfig: rotate.RotateConfig{
		Compress:         true,
		MaxFileAge:       0,
		MaxFileCount: 100,
		MaxFileSize:      1024,
	},
	Compress:true,
	Log400Message: true,
	LogInvalidMessage: true,
	LogInvalid200Message: true,
	LogInvalid200RetryMessage: true,
	Log429RetryMessage: true,
	RetryDelayInSeconds: 1,
	RejectDelayInSeconds: 1,
	MaxRejectRetryTimes: 3,
	MaxRetryTimes: 3,
	MaxRequestBodySize: 256,
	MaxResponseBodySize: 256,
}

type BulkProcessor struct {
	RotateConfig              rotate.RotateConfig
	Compress                  bool
	Log400Message             bool
	LogInvalidMessage         bool
	LogInvalid200Message      bool
	LogInvalid200RetryMessage bool
	Log429RetryMessage        bool
	RetryDelayInSeconds       int
	RejectDelayInSeconds      int
	MaxRejectRetryTimes       int //max_reject_retry_times
	MaxRetryTimes             int //max_reject_times
	MaxRequestBodySize        int
	MaxResponseBodySize       int
}

func (joint *BulkProcessor) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data []byte,httpClient *fasthttp.Client) (int, bool) {
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
	//compress := joint.GetBool("compress", true)

	req := fasthttp.AcquireRequest()
	req.Reset()
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("bulk_indexing")

	if joint.Compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	req.Header.SetContentType("application/x-ndjson")

	if cfg.BasicAuth != nil {
		req.URI().SetUsername(cfg.BasicAuth.Username)
		req.URI().SetPassword(cfg.BasicAuth.Password)
	}

	if len(data) > 0 {
		if joint.Compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			req.SetBody(data)

			//buggy
			//req.SetBodyStreamWriter(func(w *bufio.Writer) {
			//	_,err:=w.Write(data)
			//	if err!=nil{
			//		log.Error(err)
			//	}
			//	err=w.Flush()
			//	if err!=nil{
			//		log.Error(err)
			//	}
			//})
		}

		if req.GetBodyLength()<=0{
			log.Error("INIT: after set, but body is zero,",len(data),",is compress:",joint.Compress)
		}
	}else{
		log.Error("INIT: data length is zero,",string(data),",is compress:",joint.Compress)
	}
	retryTimes := 0

DO:

	if req.GetBodyLength()<=0{
		log.Error("DO: data length is zero,",string(data),",is compress:",joint.Compress)
	}

	if cfg.TrafficControl != nil {
	RetryRateLimit:

		if cfg.TrafficControl.MaxQpsPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(cfg.Name, endpoint+"max_qps", int(cfg.TrafficControl.MaxQpsPerNode)).Allow() {
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}

		if cfg.TrafficControl.MaxBytesPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(cfg.Name, endpoint+"max_bps", int(cfg.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(), req.GetRequestLength()) {
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}

	}

	err := httpClient.Do(req, resp)

	if resp == nil {
		if global.Env().IsDebug {
			log.Error(err)
		}
		return 0, false
	}

	// Do we need to decompress the response?
	var resbody = resp.GetRawBody()
	if global.Env().IsDebug {
		log.Trace(resp.StatusCode(), string(util.EscapeNewLine(resbody)))
	}

	if err != nil {
		log.Error("status:", resp.StatusCode(), ",", endpoint, ",",err," ", util.SubString(string(util.EscapeNewLine(resbody)), 0, 256))
		return resp.StatusCode(), false
	}

	if resp.StatusCode() == 400 {

		if joint.Log400Message {

			bodyString := string(resbody)
			if rate.GetRateLimiter("log_400_message", endpoint, 1, 1, 5*time.Second).Allow() {
				log.Warn("status:", resp.StatusCode(), ",",req.URI().String(),",data length:",len(data) ,",data:",util.SubString(util.UnsafeBytesToString(util.EscapeNewLine(data)), 0, 256),",req body:",util.SubString(util.UnsafeBytesToString(util.EscapeNewLine(req.GetRawBody())), 0, 256),",", util.SubString(bodyString, 0, 256))
			}

			logPath := path.Join(global.Env().GetLogDir(), cfg.Name, "400", "requests.log")
			logHandler := rotate.GetFileHandler(logPath, joint.RotateConfig)

			logHandler.WriteBytesArray(
				[]byte("\nURL:"),
				[]byte(url),
				[]byte("\nRequest:\n"),
				[]byte(util.SubString(string(util.EscapeNewLine(data)), 0, joint.MaxRequestBodySize)),
				[]byte("\nResponse:\n"),
				[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxRequestBodySize)),
			)
		}
		return resp.StatusCode(), false
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {
		if resp.StatusCode() == http.StatusOK {
			//200{"took":2,"errors":true,"items":[
			hit := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
			if hit {
				if joint.LogInvalid200Message {
					if rate.GetRateLimiter("log_invalid_200_message", endpoint, 1, 1, 5*time.Second).Allow() {
						log.Warn("status:", resp.StatusCode(), ",", endpoint, ",", util.SubString(string(util.EscapeNewLine(resbody)), 0, 256))
					}

					logPath := path.Join(global.Env().GetLogDir(), cfg.Name, "invalid_200", "requests.log")
					logHandler := rotate.GetFileHandler(logPath, joint.RotateConfig)

					logHandler.WriteBytesArray(
						[]byte("\nURL:"),
						[]byte(url),
						[]byte("\nRequest:\n"),
						[]byte(util.SubString(string(util.EscapeNewLine(data)), 0, joint.MaxRequestBodySize)),
						[]byte("\nResponse:\n"),
						[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxRequestBodySize)),
					)
				}
				delayTime := joint.RetryDelayInSeconds
				if delayTime <= 0 {
					delayTime = 10
				}
				if joint.MaxRetryTimes <= 0 {
					joint.MaxRetryTimes = 3
				}
				if retryTimes >= joint.MaxRetryTimes {
					log.Errorf("invalid 200, retried %v times, quit retry", retryTimes)
					return resp.StatusCode(), false
				}
				time.Sleep(time.Duration(delayTime) * time.Second)
				log.Debugf("invalid 200, retried %v times, will try again", retryTimes)
				retryTimes++
				goto DO
			}
		}
		return resp.StatusCode(), true
	} else if resp.StatusCode() == 429 {
		delayTime := joint.RejectDelayInSeconds
		if delayTime <= 0 {
			delayTime = 5
		}
		time.Sleep(time.Duration(delayTime) * time.Second)
		if joint.MaxRejectRetryTimes <= 0 {
			joint.MaxRejectRetryTimes = 12 //1min
		}
		if retryTimes >= joint.MaxRejectRetryTimes {
			log.Errorf("rejected 429, retried %v times, quit retry", retryTimes)
			return resp.StatusCode(), false
		}
		log.Debugf("rejected 429, retried %v times, will try again", retryTimes)
		retryTimes++
		goto DO
	} else {

		if joint.LogInvalidMessage {
			bodyString := string(resbody)
			if rate.GetRateLimiter("log_invalid_messages", endpoint, 1, 1, 5*time.Second).Allow() {
				log.Warn("status:", resp.StatusCode(), ",", endpoint, ",", util.SubString(bodyString, 0, 256))
			}

			logPath := path.Join(global.Env().GetLogDir(), cfg.Name, "invalid", "requests.log")
			logHandler := rotate.GetFileHandler(logPath, joint.RotateConfig)

			logHandler.WriteBytesArray(
				[]byte("\nURL:"),
				[]byte(url),
				[]byte("\nRequest:\n"),
				[]byte(util.SubString(string(util.EscapeNewLine(data)), 0, joint.MaxRequestBodySize)),
				[]byte("\nResponse:\n"),
				[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxRequestBodySize)),
			)
		}

		return resp.StatusCode(), false
	}
	return resp.StatusCode(), true
}
