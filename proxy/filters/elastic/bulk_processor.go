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
	"net/http"
	"path"
	"strings"
	"time"
	pool "github.com/libp2p/go-buffer-pool"
)

var bufferPool = bytebufferpool.NewPool(65536, 655360)
var smallSizedPool = bytebufferpool.NewPool(512, 655360)

var NEWLINEBYTES = []byte("\n")
var p pool.BufferPool
func WalkBulkRequests(data []byte,docBuff []byte, eachLineFunc func(eachLine []byte) (skipNextLine bool), metaFunc func(metaBytes []byte, actionStr, index, typeName, id string) (err error), payloadFunc func(payloadBytes []byte)) (int, error) {

	nextIsMeta := true
	skipNextLineProcessing := false
	var docCount = 0

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(util.GetSplitFunc(NEWLINEBYTES))

	sizeOfDocBuffer:=len(docBuff)
	if sizeOfDocBuffer>0{
		if sizeOfDocBuffer<1024{
			log.Debug("doc buffer size maybe too small,",sizeOfDocBuffer)
		}
		scanner.Buffer(docBuff, sizeOfDocBuffer)
	}

	processedBytesCount:=0
	for scanner.Scan() {
		scannedByte := scanner.Bytes()
		bytesCount:=len(scannedByte)
		processedBytesCount+=bytesCount
		if scannedByte == nil || bytesCount <= 0 {
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
				log.Error(err)
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

	if processedBytesCount+sizeOfDocBuffer<=len(data){
		log.Warn("bulk requests was not fully processed,",processedBytesCount,"/",len(data),", you may need to increase `doc_buffer_size`, re-processing with memory inefficient way now")

		lines:=bytes.Split(data,NEWLINEBYTES)

		//reset
		nextIsMeta = true
		skipNextLineProcessing = false
		docCount = 0
		processedBytesCount=0

		for _,line:=range lines{
			bytesCount:=len(line)
			processedBytesCount+=bytesCount
			if line == nil || bytesCount <= 0 {
				log.Debug("invalid line, continue")
				continue
			}

			if eachLineFunc != nil {
				skipNextLineProcessing = eachLineFunc(line)
			}

			if skipNextLineProcessing {
				skipNextLineProcessing = false
				nextIsMeta = true
				log.Debug("skip body processing")
				continue
			}

			if nextIsMeta {
				nextIsMeta = false
				var actionStr string
				var index string
				var typeName string
				var id string
				actionStr, index, typeName, id = parseActionMeta(line)

				err := metaFunc(line, actionStr, index, typeName, id)
				if err != nil {
					log.Error(err)
					return docCount, err
				}

				docCount++

				if actionStr == actionDelete {
					nextIsMeta = true
					payloadFunc(nil)
				}
			} else {
				nextIsMeta = true
				payloadFunc(line)
			}
		}

	}


	log.Tracef("total [%v] operations in bulk requests", docCount)
	return docCount, nil
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

	SaveFailure       bool
	FailureRequestsQueue       string
	InvalidRequestsQueue       string
	DeadRequestsQueue          string
	DocBufferSize          int
}

type API_STATUS string

const SUCCESS API_STATUS = "success"
const INVALID API_STATUS = "invalid"
const PARTIAL API_STATUS = "partial"
const FAILURE API_STATUS = "failure"

func (joint *BulkProcessor) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data []byte, httpClient *fasthttp.Client) (status_code int, status API_STATUS) {

	if data == nil || len(data) == 0 {
		log.Error("data size is empty,", endpoint)
		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")

		if joint.SaveFailure{
			queue.Push(joint.FailureRequestsQueue,data)
		}

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

		if joint.SaveFailure{
			queue.Push(joint.FailureRequestsQueue,data)
		}

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

		if joint.SaveFailure{
			queue.Push(joint.FailureRequestsQueue,data)
		}

		return resp.StatusCode(), FAILURE
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		stats.Increment("elasticsearch."+cfg.Name+".bulk","200_requests")

		if resp.StatusCode() == http.StatusOK {
			//TODO verify each es version's error response
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
						[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.MaxResponseBodySize)),
					)
				}

				//decode response
				response := elastic.BulkResponse{}
				err := response.UnmarshalJSON(resbody)
				if err != nil {
					panic(err)
				}

				var contains400Error bool
				var invalidCount =0
				invalidOffset := map[int]elastic.BulkActionMetadata{}
				for i, v := range response.Items {
					item := v.GetItem()
					if item.Error != nil {
						if item.Status==400{
							contains400Error = true
							invalidCount++
						}
						//fmt.Println(i,",",item.Status,",",item.Error.Type)
						//TODO log invalid requests
						//send 400 requests to dedicated queue, the rest go to failure queue
						invalidOffset[i] = v
					}
				}

				if invalidCount>0&&invalidCount==len(response.Items){
					//all 400 error
					if joint.SaveFailure{
						queue.Push(joint.InvalidRequestsQueue, data)
					}
					return 400, INVALID
				}

				if global.Env().IsDebug{
					log.Trace("invalid requests:",invalidOffset,len(invalidOffset) , len(response.Items))
				}

				if len(invalidOffset) > 0 && len(invalidOffset) < len(response.Items) {
					requestBytes := req.GetRawBody()
					errorItems := bytebufferpool.Get()
					retryableItems := bytebufferpool.Get()

					var offset = 0
					var match = false
					var retryable = false
					var response elastic.BulkActionMetadata
					invalidCount =0
					var failureCount =0
					//walk bulk message, with invalid id, save to another list

					var docBuffer []byte
					docBuffer=p.Get(joint.DocBufferSize)
					defer p.Put(docBuffer)

					WalkBulkRequests(requestBytes,docBuffer, func(eachLine []byte) (skipNextLine bool) {
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

					if invalidCount>0{
						stats.IncrementBy("elasticsearch."+cfg.Name+".bulk","200_invalid_docs", int64(invalidCount))
					}

					if failureCount>0{
						stats.IncrementBy("elasticsearch."+cfg.Name+".bulk","200_failure_docs", int64(failureCount))
					}

					if len(invalidOffset)>0{
						stats.Increment("elasticsearch."+cfg.Name+".bulk","200_partial_requests")
					}


					if errorItems.Len() > 0 {
						if joint.SaveFailure{
							queue.Push(joint.InvalidRequestsQueue, errorItems.Bytes())
							//send to redis channel
							errorItems.Reset()
							bytebufferpool.Put(errorItems)
						}
					}

					if retryableItems.Len() > 0 {
						if joint.SaveFailure {
							queue.Push(joint.FailureRequestsQueue, retryableItems.Bytes())
							retryableItems.Reset()
							bytebufferpool.Put(retryableItems)
						}
					}

					if contains400Error {
						return 400, PARTIAL
					}
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
					if joint.SaveFailure{
						queue.Push(joint.FailureRequestsQueue,data)
					}
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
			if joint.SaveFailure{
				queue.Push(joint.FailureRequestsQueue,data)
			}
			return resp.StatusCode(), FAILURE
		}
		log.Debugf("rejected 429, retried %v times, will try again", retryTimes)
		retryTimes++
		goto DO
	}else if resp.StatusCode()==400{
		//handle 400 error
		if joint.SaveFailure{
			queue.Push(joint.InvalidRequestsQueue, data)
		}

		stats.Increment("elasticsearch."+cfg.Name+".bulk","400_requests")


		if joint.LogInvalidMessage {
			if rate.GetRateLimiter("log_invalid_messages", endpoint, 1, 1, 5*time.Second).Allow() {
				log.Warn("status:", resp.StatusCode(), ",", endpoint, ",", util.SubString(util.UnsafeBytesToString(resbody), 0, 256))
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

		return resp.StatusCode(), INVALID
	} else {

		stats.Increment("elasticsearch."+cfg.Name+".bulk","5xx_requests")

		if joint.LogInvalidMessage {
			if rate.GetRateLimiter("log_invalid_messages", endpoint, 1, 1, 5*time.Second).Allow() {
				log.Warn("status:", resp.StatusCode(), ",", endpoint, ",", util.SubString(util.UnsafeBytesToString(resbody), 0, 256))
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

		if joint.SaveFailure{
			queue.Push(joint.FailureRequestsQueue, data)
		}

		return resp.StatusCode(), FAILURE
	}

}
