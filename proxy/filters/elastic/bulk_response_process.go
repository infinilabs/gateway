/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package elastic

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
)

type BulkResponseProcess struct {
	config    *Config
	retryFlow *common.FilterFlow
}

func (this *BulkResponseProcess) Name() string {
	return "bulk_response_process"
}

func (this *BulkResponseProcess) Filter(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())
	if string(ctx.Request.Header.Method()) != "POST" || !util.ContainStr(path, "_bulk") {
		return
	}

	if ctx.Response.StatusCode() == http.StatusOK || ctx.Response.StatusCode() == http.StatusCreated {
		var resbody = ctx.Response.GetRawBody()
		requestBytes := ctx.Request.GetRawBody()

		nonRetryableItems := bytebufferpool.Get()
		retryableItems := bytebufferpool.Get()
		successItems := bytebufferpool.Get()

		containError := this.HandleBulkResponse(ctx,this.config.SafetyParse, requestBytes, resbody, this.config.DocBufferSize, nonRetryableItems, retryableItems,successItems)
		if containError {

			if global.Env().IsDebug {
				log.Error("error in bulk requests,", ctx.Response.StatusCode(), util.SubString(string(resbody), 0, this.config.MessageTruncateSize))
			}

			if nonRetryableItems.Len() > 0 && this.config.InvalidQueue!="" {
				nonRetryableItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(nonRetryableItems.Bytes(), true)
				queue.Push(queue.GetOrInitConfig(this.config.InvalidQueue), bytes)
				bytebufferpool.Put(nonRetryableItems)

				queue.Push(queue.GetOrInitConfig(this.config.InvalidQueue+"-bulk-error-messages"), util.MustToJSONBytes(
					util.MapStr{
						"request": util.MapStr{
							"uri":ctx.Request.URI().String(),
							"body":util.SubString(string(ctx.Request.GetRawBody()), 0, 1024*4),
						},
						"response": util.MapStr{
							"status": ctx.Response.StatusCode(),
							"body":util.SubString(string(ctx.Response.GetRawBody()), 0, 1024*4),
						},
					}))


				stats.IncrementBy("bulk_response","invalid_unretry_items", int64(nonRetryableItems.Len()))

				if len(this.config.TagsOnInvalid)>0{
					ctx.UpdateTags(this.config.TagsOnInvalid,nil)
				}
			}

			if retryableItems.Len() > 0&& this.config.FailureQueue!=""  {

				retryableItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(retryableItems.Bytes(), true)

				if this.config.PartialFailureRetry&&this.retryFlow!=nil{
					ctx.AddFlowProcess("retry_flow:" + this.retryFlow.ID)
					this.retryFlow.Process(ctx)
				}

				queue.Push(queue.GetOrInitConfig(this.config.FailureQueue), bytes)
				bytebufferpool.Put(retryableItems)

				stats.IncrementBy("bulk_response","failure_retry_items", int64(retryableItems.Len()))

				if len(this.config.TagsOnFailure)>0{
					ctx.UpdateTags(this.config.TagsOnFailure,nil)
				}
			}

			if successItems.Len() > 0 && this.config.SuccessQueue!="" {
				successItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(successItems.Bytes(), true)
				queue.Push(queue.GetOrInitConfig(this.config.SuccessQueue), bytes)
				bytebufferpool.Put(successItems)

				stats.IncrementBy("bulk_response","partial_success_items", int64(successItems.Len()))

				if len(this.config.TagsOnSuccess)>0{
					ctx.UpdateTags(this.config.TagsOnSuccess,nil)
				}
			}

			if successItems.Len()==0&&retryableItems.Len()==0{
				if len(this.config.TagsOnAllError)>0{
					ctx.UpdateTags(this.config.TagsOnAllError,nil)
				}
			}

			//出错不继续交由后续流程，直接结束处理
			if !this.config.ContinueOnError {
				log.Errorf("this.config.ContinueOnError:%v, %v",this.config.ContinueOnError,ctx.GetFlowProcess())
				ctx.Finished()
				return
			}
		}else{
			//没有错误，标记处理完成
			if len(this.config.TagsOnAllSuccess)>0{
				ctx.UpdateTags(this.config.TagsOnAllSuccess,nil)
			}

			if this.config.SuccessQueue!=""{
				queue.Push(queue.GetOrInitConfig(this.config.SuccessQueue), ctx.Request.Encode())
				bytebufferpool.Put(successItems)
			}

			if !this.config.ContinueOnSuccess {
				ctx.Finished()
				return
			}
		}
	}else{

		if len(this.config.TagsOnAllError)>0{
			ctx.UpdateTags(this.config.TagsOnAllError,nil)
		}

		queue.Push(queue.GetOrInitConfig(this.config.InvalidQueue+"-req-error-messages"), util.MustToJSONBytes(
			util.MapStr{
				"context": ctx.GetFlowProcess(),
				"request": util.MapStr{
					"uri":ctx.Request.URI().String(),
					"body":util.SubString(string(ctx.Request.GetRawBody()), 0, 1024*4),
				},
				"response": util.MapStr{
					"status": ctx.Response.StatusCode(),
					"body":util.SubString(string(ctx.Response.GetRawBody()), 0, 1024*4),
				},
			}))

	}
}

func (this *BulkResponseProcess) HandleBulkResponse(ctx *fasthttp.RequestCtx,safetyParse bool, requestBytes, resbody []byte, docBuffSize int, nonRetryableItems, retryableItems,successItems *bytebufferpool.ByteBuffer) bool {
	containError := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
	if containError {
		//decode response
		response := elastic.BulkResponse{}
		err := response.UnmarshalJSON(resbody)
		if err != nil {
			panic(err)
		}
		invalidOffset := map[int]elastic.BulkActionMetadata{}
		var validCount = 0
		var statsCodeStats = map[int]int{}
		for i, v := range response.Items {
			item := v.GetItem()

			x, ok := statsCodeStats[item.Status]
			if !ok {
				x = 0
			}
			x++
			statsCodeStats[item.Status] = x

			if item.Error != nil {
				invalidOffset[i] = v
			} else {
				validCount++
			}
		}
		if len(invalidOffset)>0{
			log.Debug("bulk status:", statsCodeStats)
		}

		for x, y := range statsCodeStats {
			stats.IncrementBy("bulk_items", fmt.Sprintf("%v", x), int64(y))
		}

		ctx.Set("bulk_response_status",statsCodeStats)
		ctx.Response.Header.Set("X-BulkRequest-Failed","true")

		var offset = 0
		var match = false
		var retryable = false
		var actionMetadata elastic.BulkActionMetadata
		var docBuffer []byte
		docBuffer = p.Get(docBuffSize)
		defer p.Put(docBuffer)

		WalkBulkRequests(safetyParse, requestBytes, docBuffer, func(eachLine []byte) (skipNextLine bool) {
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {
			actionMetadata, match = invalidOffset[offset]
			if match {

				//find invalid request
				if actionMetadata.GetItem().Status >= 400 && actionMetadata.GetItem().Status < 500 && actionMetadata.GetItem().Status != 429 {
					retryable = false
					//contains400Error = true
					if nonRetryableItems.Len() > 0 {
						nonRetryableItems.WriteByte('\n')
					}
					nonRetryableItems.Write(metaBytes)
				} else {
					retryable = true
					if retryableItems.Len() > 0 {
						retryableItems.WriteByte('\n')
					}
					retryableItems.Write(metaBytes)
				}
			}else {
				if successItems.Len() > 0 {
					successItems.WriteByte('\n')
				}
				successItems.Write(metaBytes)
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
			}else {
				if successItems.Len() > 0 {
					successItems.WriteByte('\n')
				}
				successItems.Write(payloadBytes)
			}
		})

	}
	return containError
}

//TODO remove
func HandleBulkResponse2(safetyParse bool, requestBytes, resbody []byte, docBuffSize int, reqBuffer *common.BulkBuffer,nonRetryableItems,retryableItems *bytebufferpool.ByteBuffer) (bool,map[int]int) {
	containError := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
	var statsCodeStats = map[int]int{}
	if containError {
		//decode response
		response := elastic.BulkResponse{}
		err := response.UnmarshalJSON(resbody)
		if err != nil {
			panic(err)
		}
		//var contains400Error = false
		invalidOffset := map[int]elastic.BulkActionMetadata{}
		var validCount = 0
		for i, v := range response.Items {
			item := v.GetItem()
			reqBuffer.SetResponseStatus(i,item.Status)

			x, ok := statsCodeStats[item.Status]
			if !ok {
				x = 0
			}
			x++
			statsCodeStats[item.Status] = x

			if item.Error != nil {
				invalidOffset[i] = v
			} else {
				validCount++
			}
		}

		if len(invalidOffset)>0{
			if global.Env().IsDebug{
				log.Debug("bulk status:", statsCodeStats)
			}
		}

		//de-dup
		for x, y := range statsCodeStats {
			stats.IncrementBy("bulk_items", fmt.Sprintf("%v", x), int64(y))
		}

		var offset = 0
		var match = false
		var retryable = false
		var actionMetadata elastic.BulkActionMetadata
		var docBuffer []byte
		docBuffer = p.Get(docBuffSize)
		defer p.Put(docBuffer)

		WalkBulkRequests(safetyParse, requestBytes, docBuffer, func(eachLine []byte) (skipNextLine bool) {
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {
			actionMetadata, match = invalidOffset[offset]
			if match {
				//find invalid request
				if actionMetadata.GetItem().Status >= 400 && actionMetadata.GetItem().Status < 500 && actionMetadata.GetItem().Status != 429 {
					retryable = false
					if nonRetryableItems.Len() > 0 {
						nonRetryableItems.WriteByte('\n')
					}
					nonRetryableItems.Write(metaBytes)
				} else {
					retryable = true
					if retryableItems.Len() > 0 {
						retryableItems.WriteByte('\n')
					}
					retryableItems.Write(metaBytes)
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

	}
	return containError,statsCodeStats
}

type Config struct {
	SafetyParse bool `config:"safety_parse"`

	DocBufferSize                       int      `config:"doc_buffer_size"`
	SuccessQueue                        string   `config:"success_queue"`
	InvalidQueue                        string   `config:"invalid_queue"`
	FailureQueue                        string   `config:"failure_queue"`
	MessageTruncateSize                 int      `config:"message_truncate_size"`
	PartialFailureRetry                 bool     `config:"partial_failure_retry"`//是否主动重试，只有部分失败的请求，避免大量没有意义的 409
	PartialFailureMaxRetryTimes         int      `config:"partial_failure_max_retry_times"`//是否主动重试，只有部分失败的请求，避免大量没有意义的 409
	PartialFailureRetryDelayLatencyInMs int      `config:"partial_failure_retry_latency_in_ms"`//是否主动重试，只有部分失败的请求，避免大量没有意义的 409
	ContinueOnError                     bool     `config:"continue_on_error"`
	ContinueOnSuccess                   bool     `config:"continue_on_success"`


	TagsOnAllSuccess                       []string `config:"tag_on_all_success"`
	TagsOnAllError                       []string `config:"tag_on_all_error"`

	TagsOnSuccess                       []string `config:"tag_on_success"`
	//TagsOnError                         []string `config:"tag_on_error"`
	TagsOnPartial                       []string `config:"tag_on_partial"`
	TagsOnFailure                       []string `config:"tag_on_failure"`
	TagsOnInvalid                       []string `config:"tag_on_invalid"`

	RetryFlow  string `config:"retry_flow"`

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_response_process", NewBulkResponseValidate,&Config{})
}

func NewBulkResponseValidate(c *config.Config) (pipeline.Filter, error) {
	cfg := Config{
		DocBufferSize: 256 * 1024,
		SafetyParse:   true,
		MessageTruncateSize:   1024,
		ContinueOnError: false,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner := BulkResponseProcess{config: &cfg}

	if runner.config.RetryFlow!=""&&runner.config.PartialFailureRetry{
		flow := common.MustGetFlow(runner.config.RetryFlow)
		runner.retryFlow=&flow
	}

	return &runner, nil
}
