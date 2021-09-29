/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package elastic

import (
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"net/http"
)

type BulkResponseValidate struct {
	param.Parameters
}

func (this BulkResponseValidate) Name() string {
	return "bulk_response_validate"
}

func (this BulkResponseValidate) Process(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())
	if string(ctx.Request.Header.Method()) != "POST" || !util.ContainStr(path, "_bulk") {
		return
	}

	if ctx.Response.StatusCode() == http.StatusOK || ctx.Response.StatusCode() == http.StatusCreated {
		var resbody = ctx.Response.GetRawBody()
		containError, err := jsonparser.GetBoolean(resbody, "errors")
		if containError && err == nil {
			if global.Env().IsDebug {
				log.Error("error in bulk requests,", util.SubString(string(resbody), 0, 256))
			}

			//decode response
			response := elastic.BulkResponse{}
			err := response.UnmarshalJSON(resbody)
			if err != nil {
				panic(err)
			}
			var contains400Error = false
			//busyRejectOffset := map[int]elastic.BulkActionMetadata{}
			invalidOffset := map[int]elastic.BulkActionMetadata{}
			//failureOffset := map[int]elastic.BulkActionMetadata{}
			var invalidCount =0
			for i, v := range response.Items {
				item := v.GetItem()
				if item.Error != nil {
					invalidCount++
					invalidOffset[i]=v
				}
			}

			if invalidCount>0{

				requestBytes := ctx.Request.GetRawBody()
				nonRetryableItems := bytebufferpool.Get()
				retryableItems := bytebufferpool.Get()

				var offset = 0
				var match = false
				var retryable = false
				var response elastic.BulkActionMetadata
				invalidCount = 0
				var failureCount = 0
				//walk bulk message, with invalid id, save to another list

				var docBuffer []byte
				docBuffer = p.Get(this.GetIntOrDefault("doc_buffer_size",256*1024))
				defer p.Put(docBuffer)

				WalkBulkRequests(requestBytes, docBuffer, func(eachLine []byte) (skipNextLine bool) {
					return false
				}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {
					response, match = invalidOffset[offset]
					if match {

						//switch response.GetItem().Status {
						//case 0:
						//	//network connection issue
						//	//failureOffset[i] = v
						//	retryableItems.Write(metaBytes)
						//case 429:
						//	retryableItems.Write(metaBytes)
						//	break
						//default:
						//	if item.Status>= 400 && item.Status<500{
						//		invalidOffset[i] = v
						//	}else if item.Status>=500{
						//		failureOffset[i] = v
						//	}else{
						//		//assume they are successful requests
						//	}
						//}

						//find invalid request
						if response.GetItem().Status >= 400 && response.GetItem().Status < 500 && response.GetItem().Status != 429 {
							retryable = false
							contains400Error = true
							if nonRetryableItems.Len() > 0 {
								nonRetryableItems.WriteByte('\n')
							}
							nonRetryableItems.Write(metaBytes)
							invalidCount++
						} else {
							retryable = true
							if retryableItems.Len() > 0 {
								retryableItems.WriteByte('\n')
							}
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
								if retryableItems.Len() > 0 {
									retryableItems.WriteByte('\n')
								}
								retryableItems.Write(payloadBytes)
							} else {
								if nonRetryableItems.Len() > 0 {
									nonRetryableItems.WriteByte('\n')
								}
								nonRetryableItems.Write(payloadBytes)
							}
						}
					}
				})

				//if invalidCount > 0 {
				//	stats.IncrementBy("elasticsearch."+meta.Config.Name+".bulk", "200_invalid_docs", int64(invalidCount))
				//}
				//
				//if failureCount > 0 {
				//	stats.IncrementBy("elasticsearch."+meta.Config.Name+".bulk", "200_failure_docs", int64(failureCount))
				//}
				//
				//if len(invalidOffset) > 0 {
				//	stats.Increment("elasticsearch."+meta.Config.Name+".bulk", "200_partial_requests")
				//}

				if nonRetryableItems.Len() > 0 {
					//if joint.Config.SaveFailure {
						nonRetryableItems.WriteByte('\n')
						bytes:=ctx.Request.OverrideBodyEncode(nonRetryableItems.Bytes())
						queue.Push(this.MustGetString("invalid_queue"),bytes)
						//send to redis channel
						nonRetryableItems.Reset()
						bytebufferpool.Put(nonRetryableItems)
					//}
				}

				if retryableItems.Len() > 0 {
					//if joint.Config.SaveFailure {
						retryableItems.WriteByte('\n')
						bytes:=ctx.Request.OverrideBodyEncode( retryableItems.Bytes())
						queue.Push(this.MustGetString("failure_queue"),bytes)
						retryableItems.Reset()
						bytebufferpool.Put(retryableItems)
					//}
				}

				//if global.Env().IsDebug {
				//	log.Tracef("bulk requests failed: [%v] retryable, [%v] non_retryable, in [%v] total requests", len(ret),len(invalidOffset), len(failureOffset), len(response.Items))
				//}

				//if contains400Error {
				//	return 400, PARTIAL
				//}
			}

			//delayTime := joint.Config.RetryDelayInSeconds
			//if delayTime <= 0 {
			//	delayTime = 10
			//}
			//if joint.Config.MaxRetryTimes <= 0 {
			//	joint.Config.MaxRetryTimes = 3
			//}
			//
			//if retryTimes >= joint.Config.MaxRetryTimes {
			//	log.Errorf("invalid 200, retried %v times, quit retry", retryTimes)
			//	if joint.Config.SaveFailure {
			//		queue.Push(joint.Config.FailureRequestsQueue, data)
			//	}
			//	return resp.StatusCode(), FAILURE
			//}
			//
			//time.Sleep(time.Duration(delayTime) * time.Second)
			//log.Debugf("invalid 200, retried %v times, will try again", retryTimes)
			//retryTimes++
			//goto DO
			//return resp.StatusCode(), SUCCESS

			if contains400Error{
				ctx.Response.SetStatusCode(this.GetIntOrDefault("invalid_status", 400))
			}else{
				ctx.Response.SetStatusCode(this.GetIntOrDefault("failure_status", 500))
			}

			if this.GetBool("continue_on_error",false){
				ctx.Finished()
			}

		}
	}

	//else if ctx.Response.StatusCode() == 429 {
	//
	//	delayTime := this.Config.RejectDelayInSeconds
	//	if delayTime <= 0 {
	//		delayTime = 5
	//	}
	//	time.Sleep(time.Duration(delayTime) * time.Second)
	//	if joint.Config.MaxRejectRetryTimes <= 0 {
	//		joint.Config.MaxRejectRetryTimes = 12 //1min
	//	}
	//	if retryTimes >= joint.Config.MaxRejectRetryTimes {
	//		log.Errorf("rejected 429, retried %v times, quit retry", retryTimes)
	//		if joint.Config.SaveFailure {
	//			queue.Push(joint.Config.FailureRequestsQueue, data)
	//		}
	//		return resp.StatusCode(), FAILURE
	//	}
	//	log.Debugf("rejected 429, retried %v times, will try again", retryTimes)
	//	retryTimes++
	//	goto DO
	//} else if ctx.Response.StatusCode() == 400 {
	//	//handle 400 error
	//	if joint.Config.SaveFailure {
	//		queue.Push(joint.Config.InvalidRequestsQueue, data)
	//	}
	//
	//	stats.Increment("elasticsearch."+meta.Config.Name+".bulk", "400_requests")
	//
	//	if joint.Config.LogInvalidMessage {
	//		if rate.GetRateLimiter("log_invalid_messages", host, 1, 1, 5*time.Second).Allow() {
	//			log.Warn("status:", resp.StatusCode(), ",", host, ",", util.SubString(util.UnsafeBytesToString(resbody), 0, 256))
	//		}
	//
	//		logPath := path.Join(global.Env().GetLogDir(), meta.Config.Name, "invalid", "requests.log")
	//		logHandler := rotate.GetFileHandler(logPath, joint.RotateConfig)
	//
	//		logHandler.WriteBytesArray(
	//			[]byte("\nURL:"),
	//			[]byte(url),
	//			[]byte("\nRequest:\n"),
	//			[]byte(util.SubString(string(util.EscapeNewLine(data)), 0, joint.Config.MaxRequestBodySize)),
	//			[]byte("\nResponse:\n"),
	//			[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.Config.MaxResponseBodySize)),
	//		)
	//	}
	//
	//	return resp.StatusCode(), INVALID
	//} else {
	//
	//	stats.Increment("elasticsearch."+meta.Config.Name+".bulk", "5xx_requests")
	//
	//	if joint.Config.LogInvalidMessage {
	//		if rate.GetRateLimiter("log_invalid_messages", host, 1, 1, 5*time.Second).Allow() {
	//			log.Warn("status:", resp.StatusCode(), ",", host, ",", util.SubString(util.UnsafeBytesToString(resbody), 0, 256))
	//		}
	//
	//		logPath := path.Join(global.Env().GetLogDir(), meta.Config.Name, "invalid", "requests.log")
	//		logHandler := rotate.GetFileHandler(logPath, joint.RotateConfig)
	//
	//		logHandler.WriteBytesArray(
	//			[]byte("\nURL:"),
	//			[]byte(url),
	//			[]byte("\nRequest:\n"),
	//			[]byte(util.SubString(string(util.EscapeNewLine(data)), 0, joint.Config.MaxRequestBodySize)),
	//			[]byte("\nResponse:\n"),
	//			[]byte(util.SubString(string(util.EscapeNewLine(resbody)), 0, joint.Config.MaxResponseBodySize)),
	//		)
	//	}
	//
	//	if joint.Config.SaveFailure {
	//		queue.Push(joint.Config.FailureRequestsQueue, data)
	//	}
	//
	//	return ctx.Response.StatusCode(), FAILURE
	//}

}
