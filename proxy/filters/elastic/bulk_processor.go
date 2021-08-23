package elastic

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
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
	"time"
)

var bufferPool = bytebufferpool.NewPool(65536, 655360)
var smallSizedPool = bytebufferpool.NewPool(512, 655360)

var NEWLINEBYTES = []byte("\n")

func WalkBulkRequests(data []byte, eachLineFunc func(eachLine []byte) (skipNextLine bool), metaFunc func(metaBytes []byte, actionStr, index, typeName, id string) (err error), payloadFunc func(payloadBytes []byte)) (int, error) {

	nextIsMeta := true
	skipNextLineProcessing := false

	var docCount = 0

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(util.GetSplitFunc(NEWLINEBYTES))

	for scanner.Scan() {
		scannedByte := scanner.Bytes()
		if scannedByte == nil || len(scannedByte) <= 0 {
			log.Debug("invalid scanned byte, continue")
			continue
		}

		if eachLineFunc != nil {
			skipNextLineProcessing = eachLineFunc(scannedByte)
		}

		if skipNextLineProcessing {
			skipNextLineProcessing = false
			nextIsMeta = true
			log.Debug("skip body processing")
			continue
		}

		if nextIsMeta {

			nextIsMeta = false

			//TODO improve poor performance
			var actionStr string
			var index string
			var typeName string
			var id string
			actionStr, index, typeName, id = parseActionMeta(scannedByte)

			err := metaFunc(scannedByte, actionStr, index, typeName, id)
			if err != nil {
				return docCount, err
			}

			docCount++

			if actionStr == actionDelete {
				nextIsMeta = true
				payloadFunc(nil)
			}
		} else {
			nextIsMeta = true
			payloadFunc(scannedByte)
		}
	}

	log.Tracef("total [%v] operations in bulk requests", docCount)
	return docCount, nil
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

		metadata := elastic.GetMetadata(clusterName)
		if metadata == nil {
			if rate.GetRateLimiter("cluster_metadata", clusterName, 1, 1, 5*time.Second).Allow() {
				log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle", clusterName)
			}
			time.Sleep(10 * time.Second)
			return
		}

		esConfig := elastic.GetConfig(clusterName)

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

		docCount, err := WalkBulkRequests(body, func(eachLine []byte) (skipNextLine bool) {
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
				fmt.Println("generating ID:", actionStr)
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
				metadata = elastic.GetMetadata(clusterName)
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
			log.Error(err)
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
				Compress:                  this.GetBool("compress", false),
				LogInvalidMessage:         this.GetBool("log_invalid_message", true),
				LogInvalid200Message:      this.GetBool("log_invalid_200_message", true),
				LogInvalid200RetryMessage: this.GetBool("log_200_retry_message", true),
				Log429RetryMessage:        this.GetBool("log_429_retry_message", true),
				RetryDelayInSeconds:       this.GetIntOrDefault("retry_delay_in_second", 1),
				RejectDelayInSeconds:      this.GetIntOrDefault("reject_retry_delay_in_second", 1),
				MaxRejectRetryTimes:       this.GetIntOrDefault("max_reject_retry_times", 3),
				MaxRetryTimes:             this.GetIntOrDefault("max_retry_times", 3),
				MaxRequestBodySize:        this.GetIntOrDefault("max_logged_request_body_size", 1024),
				MaxResponseBodySize:       this.GetIntOrDefault("max_logged_response_body_size", 1024),

				FailureRequestsQueue:       this.GetStringOrDefault("failure_queue",fmt.Sprintf("%v-failure",clusterName)),
				InvalidRequestsQueue:       this.GetStringOrDefault("invalid_queue",fmt.Sprintf("%v-invalid",clusterName)),
				DeadRequestsQueue:       	this.GetStringOrDefault("dead_queue",fmt.Sprintf("%v-dead",clusterName)),
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
				code, status := bulkProcessor.Bulk(esConfig, endpoint, data, fastHttpClient)
				stats.Timing("elasticsearch."+esConfig.Name+".bulk","elapsed_ms",time.Since(start).Milliseconds())
				switch status {
				case SUCCESS:
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

func getUrlLevelBulkMeta(pathStr string) (urlLevelIndex, urlLevelType string) {

	if !util.SuffixStr(pathStr, "_bulk") {
		return urlLevelIndex, urlLevelType
	}

	if strings.Contains(pathStr, "//") {
		pathStr = strings.ReplaceAll(pathStr, "//", "/")
	}

	pathArray := strings.FieldsFunc(pathStr, func(c rune) bool {
		return c == '/'
	})

	switch len(pathArray) {
	case 3:
		urlLevelIndex = pathArray[0]
		urlLevelType = pathArray[1]
		break
	case 2:
		urlLevelIndex = pathArray[0]
		break
	}

	return urlLevelIndex, urlLevelType
}

var startPart = []byte("{\"took\":0,\"errors\":false,\"items\":[")
var itemPart = []byte("{\"index\":{\"_index\":\"fake-index\",\"_type\":\"doc\",\"_id\":\"1\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":1,\"successful\":1,\"failed\":0},\"_seq_no\":1,\"_primary_term\":1,\"status\":200}}")
var endPart = []byte("]}")

//TODO 提取出来，作为公共方法，和 indexing/bulking_indexing 的方法合并
var fastHttpClient = &fasthttp.Client{
	MaxConnDuration:     0,
	MaxIdleConnDuration: 0,
	ReadTimeout:         time.Second * 60,
	WriteTimeout:        time.Second * 60,
	TLSConfig:           &tls.Config{InsecureSkipVerify: true},
}

type BulkProcessor struct {
	RotateConfig              rotate.RotateConfig
	Compress                  bool
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

	FailureRequestsQueue       string
	InvalidRequestsQueue       string
	DeadRequestsQueue          string
}

type API_STATUS string

const SUCCESS API_STATUS = "success"
const PARTIAL API_STATUS = "partial_success"
const FAILURE API_STATUS = "failure"

func (joint *BulkProcessor) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data []byte, httpClient *fasthttp.Client) (status_code int, status API_STATUS) {

	if data == nil || len(data) == 0 {
		log.Error("data size is empty,", endpoint)
		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")
		return 0, FAILURE
	}

	if cfg.IsTLS() {
		endpoint = "https://" + endpoint
	} else {
		endpoint = "http://" + endpoint
	}

	url := fmt.Sprintf("%s/_bulk", endpoint)

	req := fasthttp.AcquireRequest()
	req.Reset()
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("_bulk")

	//TODO handle response, if client not support gzip, return raw body
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
		}

		if req.GetBodyLength() <= 0 {
			log.Error("INIT: after set, but body is zero,", len(data), ",is compress:", joint.Compress)
		}
	} else {
		log.Error("INIT: data length is zero,", string(data), ",is compress:", joint.Compress)
	}
	retryTimes := 0

DO:

	if req.GetBodyLength() <= 0 {
		log.Error("DO: data length is zero,", string(data), ",is compress:", joint.Compress)
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
		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")
		return 0, FAILURE
	}

	// Do we need to decompress the response?
	var resbody = resp.GetRawBody()
	if global.Env().IsDebug {
		log.Trace(resp.StatusCode(), string(util.EscapeNewLine(resbody)))
	}

	if err != nil {
		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")

		log.Error("status:", resp.StatusCode(), ",", endpoint, ",", err, " ", util.SubString(string(util.EscapeNewLine(resbody)), 0, 256))
		return resp.StatusCode(), FAILURE
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		stats.Increment("elasticsearch."+cfg.Name+".bulk","200_requests")

		if resp.StatusCode() == http.StatusOK {
			//TODO verify each es version's error response
			hit := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
			if hit {
				//decode response
				response := elastic.BulkResponse{}
				err := response.UnmarshalJSON(resbody)
				if err != nil {
					panic(err)
				}

				invalidOffset := map[int]elastic.BulkActionMetadata{}
				for i, v := range response.Items {
					item := v.GetItem()
					if item.Error != nil {
						//fmt.Println(i,",",item.Status,",",item.Error.Type)
						//TODO log invalid requests
						//send 400 requests to dedicated queue, the rest go to failure queue
						invalidOffset[i] = v
					}
				}
				//fmt.Println("invalid requests:",invalidOffset)

				var contains400Error bool
				if len(invalidOffset) > 0 && len(invalidOffset) < len(response.Items) {
					requestBytes := req.GetRawBody()
					errorItems := bytebufferpool.Get()
					retryableItems := bytebufferpool.Get()

					var offset = 0
					var match = false
					var retryable = false
					var response elastic.BulkActionMetadata
					var invalidCount =0
					var failureCount =0
					//walk bulk message, with invalid id, save to another list
					WalkBulkRequests(requestBytes, func(eachLine []byte) (skipNextLine bool) {
						return false
					}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {

						response, match = invalidOffset[offset]
						if match {
							//find invalid request
							//fmt.Println(offset,"invalid request:",string(metaBytes),"invalid response:",response.GetItem().Result,response.GetItem().Index,response.GetItem().Type,response.GetItem().Index,response.GetItem().ID,response.GetItem().Error)
							if response.GetItem().Status >= 400 && response.GetItem().Status < 500 && response.GetItem().Status != 429 {
								retryable = false
								contains400Error = true
								errorItems.Write(metaBytes)
								invalidCount++
							} else {
								retryable = true
								retryableItems.Write(metaBytes)
								failureCount++
							}
						}
						offset++
						return nil
					}, func(payloadBytes []byte) {
						if match {
							if payloadBytes != nil && len(payloadBytes) > 0 {
								if retryable {
									retryableItems.Write(payloadBytes)
								} else {
									errorItems.Write(payloadBytes)
								}
							}
						}
					})

					stats.IncrementBy("elasticsearch."+cfg.Name+".bulk","200_invalid_docs", int64(invalidCount))
					stats.IncrementBy("elasticsearch."+cfg.Name+".bulk","200_failure_docs", int64(failureCount))

					if errorItems.Len() > 0 {
						queue.Push(joint.InvalidRequestsQueue, errorItems.Bytes())
						//send to redis channel
						errorItems.Reset()
						bytebufferpool.Put(errorItems)
					}

					if retryableItems.Len() > 0 {
						queue.Push(joint.FailureRequestsQueue, retryableItems.Bytes())
						retryableItems.Reset()
						bytebufferpool.Put(retryableItems)
					}

					if contains400Error {
						return 400, PARTIAL
					}
				}

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
						[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxResponseBodySize)),
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
					return resp.StatusCode(), FAILURE
				}

				time.Sleep(time.Duration(delayTime) * time.Second)
				log.Debugf("invalid 200, retried %v times, will try again", retryTimes)
				retryTimes++
				goto DO
			}
		}
		return resp.StatusCode(), SUCCESS
	} else if resp.StatusCode() == 429 {
		stats.Increment("elasticsearch."+cfg.Name+".bulk","429_requests")

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
			return resp.StatusCode(), FAILURE
		}
		log.Debugf("rejected 429, retried %v times, will try again", retryTimes)
		retryTimes++
		goto DO
	} else {

		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")

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
				[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxResponseBodySize)),
			)
		}

		queue.Push(joint.FailureRequestsQueue, data)

		return resp.StatusCode(), FAILURE
	}

}
