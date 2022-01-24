package elastic

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type ElasticsearchBulkRequestMutate struct {
	DefaultIndex     string            `config:"default_index"`
	DefaultType      string            `config:"default_type"`
	FixNilType       bool              `config:"fix_null_type"`
	FixNilID         bool              `config:"fix_null_id"`
	AddTimestampToID bool              `config:"generate_enhanced_id"`
	SafetyParse      bool              `config:"safety_parse"`
	DocBufferSize    int               `config:"doc_buffer_size"`
	IndexNameRename  map[string]string `config:"index_rename"`
	TypeNameRename   map[string]string `config:"type_rename"`
}

func (filter *ElasticsearchBulkRequestMutate) Name() string {
	return "bulk_request_mutate"
}

func (this *ElasticsearchBulkRequestMutate) Filter(ctx *fasthttp.RequestCtx) {

	pathStr := util.UnsafeBytesToString(ctx.URI().Path())

	if util.SuffixStr(pathStr, "/_bulk") {
		body := ctx.Request.GetRawBody()
		var bulkBuff *bytebufferpool.ByteBuffer = bufferPool.Get()
		actionMeta := smallSizedPool.Get()
		defer smallSizedPool.Put(actionMeta)

		var docBuffer []byte
		docBuffer = p.Get(this.DocBufferSize) //doc buffer for bytes scanner
		defer p.Put(docBuffer)

		docCount, err := WalkBulkRequests(this.SafetyParse, body, docBuffer, func(eachLine []byte) (skipNextLine bool) {
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id string) (err error) {

			metaStr := util.UnsafeBytesToString(metaBytes)

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

			if (actionStr == actionIndex || actionStr == actionCreate) && len(id) == 0 && this.FixNilID {
				randID := util.GetUUID()
				if this.AddTimestampToID {
					idNew = fmt.Sprintf("%v-%v-%v", randID, time.Now().UnixNano(), util.PickRandomNumber(10))
				} else {
					idNew = randID
				}
				id = idNew
				if global.Env().IsDebug {
					log.Trace("generated new id: ", id, " for: ", metaStr)
				}
			}

			if typeNew == "" && this.FixNilType && this.DefaultType != "" {
				typeNew = this.DefaultType
				if global.Env().IsDebug {
					log.Trace("use default type: ", this.DefaultType, " for: ", metaStr)
				}
			}

			if index != "" && len(this.IndexNameRename) > 0 {
				v, ok := this.IndexNameRename[index]
				if ok {
					index = v
					indexNew = v
				} else {
					v, ok := this.IndexNameRename["*"]
					if ok {
						index = v
						indexNew = v
					}
				}
			}

			if typeName != "" && len(this.TypeNameRename) > 0 {
				v, ok := this.TypeNameRename[typeName]
				if ok {
					typeNew = v
					typeName = v
				} else {
					v, ok := this.TypeNameRename["*"]
					if ok {
						typeNew = v
						typeName = v
					}
				}
			}

			//update metadata changes
			if indexNew != "" || typeNew != "" || idNew != "" {
				var err error
				metaBytes, err = updateJsonWithNewIndex(actionStr, metaBytes, indexNew, typeNew, idNew)
				if err != nil {
					panic(err)
				}
				if global.Env().IsDebug {
					log.Trace("updated action meta,", id, ",", metaStr, "->", string(metaBytes))
				}
			}

			if actionStr == "" || index == "" || id == "" {
				log.Warn("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", metaStr)
				return errors.Error("invalid bulk action:", actionStr, ",index:", string(index), ",id:", string(id), ",", metaStr)
			}

			if global.Env().IsDebug {
				log.Tracef("final path: %s/%s/%s", index, typeName, id)
			}

			if global.Env().IsDebug {
				log.Tracef("metadata:\n%v", string(metaBytes))
			}
			actionMeta.Write(metaBytes)

			return nil
		}, func(payloadBytes []byte) {

			if actionMeta.Len() > 0 {

				if global.Env().IsDebug {
					log.Tracef("payload:\n%v", string(payloadBytes))
				}

				if bulkBuff.Len() > 0 {
					bulkBuff.Write(NEWLINEBYTES)
				}

				bulkBuff.Write(actionMeta.Bytes())
				if payloadBytes != nil && len(payloadBytes) > 0 {
					bulkBuff.Write(NEWLINEBYTES)
					bulkBuff.Write(payloadBytes)
				}

				actionMeta.Reset()
			}
		})

		if err != nil {
			log.Errorf("processing: %v docs, err: %v", docCount, err)
			return
		}

		if bulkBuff.Len() > 0 {
			bulkBuff.Write(NEWLINEBYTES)
			ctx.Request.SetRawBody(bulkBuff.Bytes())
		}
	}

}

func init() {
	pipeline.RegisterFilterPlugin("bulk_request_mutate",NewElasticsearchBulkRequestMutateFilter)
}

func NewElasticsearchBulkRequestMutateFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchBulkRequestMutate{
		FixNilID:      true,
		DocBufferSize: 256 * 1024,
		SafetyParse:   true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
