package elastic

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	pool "github.com/libp2p/go-buffer-pool"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
	"strings"
	"time"
)

var bufferPool = bytebufferpool.NewPool(65536, 655360)
var smallSizedPool = bytebufferpool.NewPool(512, 655360)

var NEWLINEBYTES = []byte("\n")
var p pool.BufferPool

func WalkBulkRequests(safetyParse bool,data []byte, docBuff []byte, eachLineFunc func(eachLine []byte) (skipNextLine bool), metaFunc func(metaBytes []byte, actionStr, index, typeName, id string) (err error), payloadFunc func(payloadBytes []byte)) (int, error) {

	nextIsMeta := true
	skipNextLineProcessing := false
	var docCount = 0

	START:

	if safetyParse {
		lines := bytes.Split(data, NEWLINEBYTES)
		//reset
		nextIsMeta = true
		skipNextLineProcessing = false
		docCount = 0
		for i, line := range lines {
			bytesCount := len(line)
			if line == nil || bytesCount <= 0 {
				if global.Env().IsDebug{
					log.Tracef("invalid line, continue, [%v/%v] [%v]\n%v",i,len(lines),string(line),util.PrintStringByteLines(lines))
				}
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
					log.Debug(err)
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

	if !safetyParse{
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Split(util.GetSplitFunc(NEWLINEBYTES))

		sizeOfDocBuffer := len(docBuff)
		if sizeOfDocBuffer > 0 {
			if sizeOfDocBuffer < 1024 {
				log.Debug("doc buffer size maybe too small,", sizeOfDocBuffer)
			}
			scanner.Buffer(docBuff, sizeOfDocBuffer)
		}

		processedBytesCount := 0
		for scanner.Scan() {
			scannedByte := scanner.Bytes()
			bytesCount := len(scannedByte)
			processedBytesCount += bytesCount
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
					if global.Env().IsDebug{
						log.Error(err)
					}
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

		if processedBytesCount+sizeOfDocBuffer <= len(data) {
			log.Warn("bulk requests was not fully processed,", processedBytesCount, "/", len(data), ", you may need to increase `doc_buffer_size`, re-processing with memory inefficient way now")
			return 0,errors.New("documents too big, skip processing")
			safetyParse=true
			goto START
		}
	}

	if global.Env().IsDebug{
		log.Tracef("total [%v] operations in bulk requests", docCount)
	}

	return docCount, nil
}

func getUrlLevelBulkMeta(pathStr string) (urlLevelIndex, urlLevelType string) {

	if !util.SuffixStr(pathStr, "_bulk") {
		return urlLevelIndex, urlLevelType
	}

	if !util.PrefixStr(pathStr,"/"){
		return urlLevelIndex, urlLevelType
	}

	if strings.Index(pathStr, "//")>=0 {
		pathStr = strings.ReplaceAll(pathStr, "//", "/")
	}

	if strings.LastIndex(pathStr,"/")==0{
		return urlLevelIndex, urlLevelType
	}

	pathArray:=strings.Split(pathStr,"/")

	switch len(pathArray) {
	case 4:
		urlLevelIndex = pathArray[1]
		urlLevelType = pathArray[2]
		break
	case 3:
		urlLevelIndex = pathArray[1]
		break
	}

	return urlLevelIndex, urlLevelType
}

var startPart = []byte("{\"took\":0,\"errors\":false,\"items\":[")
var itemPart = []byte("{\"index\":{\"_index\":\"fake-index\",\"_type\":\"doc\",\"_id\":\"1\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":1,\"successful\":1,\"failed\":0},\"_seq_no\":1,\"_primary_term\":1,\"status\":200}}")
var endPart = []byte("]}")

type BulkProcessorConfig struct {
	bulkSizeInByte int

	BulkSizeInKb         int    `config:"batch_size_in_kb,omitempty"`
	BulkSizeInMb         int    `config:"batch_size_in_mb,omitempty"`
	BulkMaxDocsCount     int    `config:"batch_size_in_docs,omitempty"`

	Compress                  bool `config:"compress"`
	RetryDelayInSeconds       int  `config:"retry_delay_in_seconds"`
	RejectDelayInSeconds      int  `config:"reject_retry_delay_in_seconds"`
	MaxRejectRetryTimes       int  `config:"max_reject_retry_times"`
	MaxRetryTimes             int  `config:"max_retry_times"`
	RequestTimeoutInSecond          int   `config:"request_timeout_in_second"`
	InvalidRequestsQueue    string `config:"invalid_queue"`
	DeadletterRequestsQueue string `config:"dead_letter_queue"`
	SafetyParse bool `config:"safety_parse"`
	DocBufferSize        int    `config:"doc_buffer_size"`
}

func (this *BulkProcessorConfig) GetBulkSizeInBytes() int  {

	this.bulkSizeInByte= 1048576 * this.BulkSizeInMb
	if this.BulkSizeInKb > 0 {
		this.bulkSizeInByte = 1024 * this.BulkSizeInKb
	}
	if this.bulkSizeInByte<=0{
		this.bulkSizeInByte=10*1024*1024
	}
	return this.bulkSizeInByte
}

var DefaultBulkProcessorConfig = BulkProcessorConfig{
		BulkMaxDocsCount:         1000,
		BulkSizeInMb:         10,
		Compress:  false,
		RetryDelayInSeconds:  1,
		RejectDelayInSeconds: 1,
		MaxRejectRetryTimes:  60,
		MaxRetryTimes:        3,
		DeadletterRequestsQueue: "dead_letter_queue",
		SafetyParse:          true,
		DocBufferSize:       256*1024,
		RequestTimeoutInSecond:60,
}

type BulkProcessor struct {
	Config       BulkProcessorConfig
}

type API_STATUS string


func (joint *BulkProcessor) Bulk(tag string,metadata *elastic.ElasticsearchMetadata, host string, buffer *common.BulkBuffer) (continueNext bool) {

	if buffer == nil || buffer.GetMessageSize() == 0 {
		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "empty_bulk_requests")
		return true
	}

	httpClient := metadata.GetActivePreferredHost(host)

	if metadata.IsTLS() {
		host = "https://" + host
	} else {
		host = "http://" + host
	}

	url := fmt.Sprintf("%s/_bulk", host)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("_bulk")

	req.Header.SetContentType("application/x-ndjson")

	if metadata.Config.BasicAuth != nil {
		req.URI().SetUsername(metadata.Config.BasicAuth.Username)
		req.URI().SetPassword(metadata.Config.BasicAuth.Password)
	}

	acceptGzipped := req.AcceptGzippedResponse()
	compressed := false

	data := buffer.Buffer.Bytes()

	if !req.IsGzipped() && joint.Config.Compress {

		_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data, fasthttp.CompressBestSpeed)
		if err != nil {
			panic(err)
		}

		//TODO handle response, if client not support gzip, return raw body
		req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
		req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
		compressed = true

	} else {
		req.SetBody(data)
	}

	if req.GetBodyLength() <= 0 {
		log.Error("INIT: after set, but body is zero,", len(data), ",is compress:", joint.Config.Compress)
	}

	// modify schema，align with elasticsearch's schema
	orignalSchema := string(req.URI().Scheme())
	orignalHost := string(req.URI().Host())
	if metadata.GetSchema() != orignalSchema {
		req.URI().SetScheme(metadata.GetSchema())
	}

	retryTimes := 0
	DO:

	if req.GetBodyLength() <= 0 {
		log.Error("DO: data length is zero,", string(data), ",is compress:", joint.Config.Compress)
	}

	metadata.CheckNodeTrafficThrottle(util.UnsafeBytesToString(req.Header.Host()), 1, req.GetRequestLength(), 0)

	//execute
	err := httpClient.DoTimeout(req, resp, time.Duration(joint.Config.RequestTimeoutInSecond)*time.Second)

	stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "http_request_count")

	if err != nil {
		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "5xx_requests")

		if rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"5xx_on_error", 1).Allow() {
			log.Error("status:", resp.StatusCode(), ",", host, ",", err, " ", util.SubString(string(util.EscapeNewLine(resp.GetRawBody())), 0, 256))
			time.Sleep(1 * time.Second)
		}

		return false
	}

	//restore body and header
	if !acceptGzipped && compressed {
		body := resp.GetRawBody()
		resp.SwapBody(body)
		resp.Header.Del(fasthttp.HeaderContentEncoding)
		resp.Header.Del(fasthttp.HeaderContentEncoding2)
	}

	// restore schema
	req.URI().SetScheme(orignalSchema)
	req.SetHost(orignalHost)

	if resp == nil {
		if global.Env().IsDebug {
			log.Error(err)
		}
		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "5xx_requests")
		return false
	}

	// Do we need to decompress the response?
	var resbody = resp.GetRawBody()
	if global.Env().IsDebug {
		log.Trace(resp.StatusCode(),string(util.EscapeNewLine(data)), string(util.EscapeNewLine(resbody)))
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		//如果是部分失败，应该将可以重试的做完，然后记录失败的消息再返回不继续
		if util.ContainStr(string(req.RequestURI()), "_bulk") {
			nonRetryableItems := bytebufferpool.Get()
			retryableItems := bytebufferpool.Get()
			nonRetryableItems.Reset()
			retryableItems.Reset()

			containError,statsCodeStats := HandleBulkResponse2(tag,joint.Config.SafetyParse, data, resbody, joint.Config.DocBufferSize, buffer, nonRetryableItems, retryableItems)
			if containError {

				stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_bulk_error_requests")

				if retryableItems.Len()>0{
					retryableItems.WriteByte('\n')
					data:=req.OverrideBodyEncode(retryableItems.Bytes(), true)

					delayTime := joint.Config.RejectDelayInSeconds
					if delayTime <= 0 {
						delayTime = 5
					}
					time.Sleep(time.Duration(delayTime) * time.Second)
					if joint.Config.MaxRejectRetryTimes <= 0 {
						joint.Config.MaxRejectRetryTimes = 12 //1min
					}
					if retryTimes >= joint.Config.MaxRejectRetryTimes {

						//continue retry before is back
						if !metadata.IsAvailable(){
							return false
						}

						log.Errorf("bulk partial failure, retried %v times, quit retry", retryTimes)
						queue.Push(queue.GetOrInitConfig(metadata.Config.ID+"_dead_letter_queue"), data)
						stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_bulk_error_requests_retry_dead")

						return true
					}
					log.Errorf("bulk partial failure, #%v retry", retryTimes)
					retryTimes++
					stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_bulk_error_requests_retry")
					goto DO
				}

				failureStatus:=buffer.GetMessageStatus(true)

				if len(failureStatus)>0{
					failureStatusStr:=util.JoinMapInt(failureStatus,":")
					log.Debugf("documents in failure: %v",failureStatusStr)
					//save message bytes, with metadata, set codec to wrapped bulk messages
					queue.Push(queue.GetOrInitConfig("failure_messages"), util.MustToJSONBytes(util.MapStr{
						"cluster_id":  metadata.Config.ID,
						"queue":  buffer.Queue,
						"request": util.MapStr{
							"uri":req.URI().String(),
							"body":util.SubString(string(req.GetRawBody()), 0, 1024*4),
						},
						"response": util.MapStr{
							"status": failureStatusStr,
							"body":util.SubString(string(resbody), 0, 1024*4),
						},
					}))
					log.Errorf("bulk requests failure,host:%v,status:%v,invalid:%v,failure:%v,res:%v", host, statsCodeStats, nonRetryableItems.Len(), retryableItems.Len(), util.SubString(string(resbody), 0, 1024))
				}

				//skip all failure messages
				if nonRetryableItems.Len()>0&&retryableItems.Len()==0{

					////handle 400 error
					if joint.Config.InvalidRequestsQueue!="" {
						queue.Push(queue.GetOrInitConfig(joint.Config.InvalidRequestsQueue), data)
					}

					stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_bulk_all_error_requests")
					return true
				}else{
					stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "what_else")
				}

				return false
			}else{
				stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_bulk_success_requests")
			}
		}else{
			stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "200_requests")
		}

		return true
	} else if resp.StatusCode() == 429 {
		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "429_requests")


		return false
	} else if resp.StatusCode() >= 400 && resp.StatusCode()<500{

		////handle 400 error
		if joint.Config.InvalidRequestsQueue!="" {
			queue.Push(queue.GetOrInitConfig(joint.Config.InvalidRequestsQueue), data)
		}

		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "400_requests")

		return true
	} else {

		stats.Increment("elasticsearch."+tag+"."+metadata.Config.Name+".bulk", "5xx_requests")

		//if joint.Config.SaveFailure {
		//	queue.Push(queue.GetOrInitConfig(joint.Config.FailureRequestsQueue), data)
		//}

		return false
	}

}
