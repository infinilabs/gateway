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
	"net/http"
)

type BulkResponseValidate struct {
	config *Config
}

func (this *BulkResponseValidate) Name() string {
	return "bulk_response_validate"
}

func (this *BulkResponseValidate) Filter(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())
	if string(ctx.Request.Header.Method()) != "POST" || !util.ContainStr(path, "_bulk") {
		return
	}

	//stats.Increment("bulk_validate.status", fmt.Sprintf("%v", ctx.Response.StatusCode()))

	if ctx.Response.StatusCode() == http.StatusOK || ctx.Response.StatusCode() == http.StatusCreated {
		var resbody = ctx.Response.GetRawBody()
		requestBytes := ctx.Request.GetRawBody()

		nonRetryableItems := bytebufferpool.Get()
		retryableItems := bytebufferpool.Get()
		successItems := bytebufferpool.Get()

		containError:=HandleBulkResponse(this.config.SafetyParse,requestBytes,resbody,this.config.DocBufferSize,nonRetryableItems,retryableItems,successItems)
		if containError {
			if global.Env().IsDebug {
				log.Error("error in bulk requests,", ctx.Response.StatusCode(), util.SubString(string(resbody), 0, 256))
			}

			if nonRetryableItems.Len() > 0 {
				nonRetryableItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(nonRetryableItems.Bytes(), true)
				queue.Push(queue.GetOrInitConfig(this.config.InvalidQueue), bytes)
				bytebufferpool.Put(nonRetryableItems)
			}

			if retryableItems.Len() > 0 {
				retryableItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(retryableItems.Bytes(), true)
				queue.Push(queue.GetOrInitConfig(this.config.FailureQueue), bytes)
				bytebufferpool.Put(retryableItems)
			}

			if successItems.Len()>0&& this.config.SaveSuccessDocsToQueue{
				successItems.WriteByte('\n')
				bytes := ctx.Request.OverrideBodyEncode(successItems.Bytes(), true)
				queue.Push(queue.GetOrInitConfig(this.config.PartialSuccessQueue), bytes)
				bytebufferpool.Put(successItems)
			}

			if nonRetryableItems.Len()>0 {
				ctx.Response.SetStatusCode(this.config.InvalidStatus)
			} else {
				ctx.Response.SetStatusCode(this.config.FailureStatus)
			}

			if !this.config.ContinueOnError {
				ctx.Finished()
			}
		}
	}
}

func HandleBulkResponse(safetyParse bool,requestBytes,resbody []byte,docBuffSize int,nonRetryableItems,retryableItems,successItems *bytebufferpool.ByteBuffer)(bool) {
	containError := util.LimitedBytesSearch(resbody, []byte("\"errors\":true"), 64)
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
			}else{
				validCount++
			}
		}

		log.Debug("bulk status:",statsCodeStats)

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

		WalkBulkRequests(safetyParse,requestBytes, docBuffer, func(eachLine []byte) (skipNextLine bool) {
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
			}else{
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

type Config struct {

	SafetyParse bool `config:"safety_parse"`

	DocBufferSize int `config:"doc_buffer_size"`

	SaveSuccessDocsToQueue bool   `config:"save_partial_success_requests"`

	PartialSuccessQueue           string `config:"partial_success_queue"`

	InvalidQueue    string `config:"invalid_queue"`

	FailureQueue    string `config:"failure_queue"`

	InvalidStatus   int    `config:"invalid_status"`

	FailureStatus   int    `config:"failure_status"`

	ContinueOnError bool   `config:"continue_on_error"`
}

func NewBulkResponseValidate(c *config.Config) (pipeline.Filter, error) {
	cfg := Config{
		InvalidStatus: 400,
		FailureStatus: 507,
		DocBufferSize: 256 * 1024,
		SafetyParse: true,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner := BulkResponseValidate{config: &cfg}

	return &runner, nil
}
