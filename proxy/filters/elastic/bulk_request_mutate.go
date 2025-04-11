// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package elastic

import (
	"fmt"
	"src/github.com/savsgio/gotils/bytes"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
)

type ElasticsearchBulkRequestMutate struct {
	DefaultIndex     string            `config:"default_index"`
	DefaultType      string            `config:"default_type"`
	FixNilType       bool              `config:"fix_null_type"`
	FixNilID         bool              `config:"fix_null_id"`
	Pipeline         string            `config:"pipeline"`
	RemoveTypeMeta   bool              `config:"remove_type"`
	RemovePipeline   bool              `config:"remove_pipeline"`
	AddTimestampToID bool              `config:"generate_enhanced_id"`
	IndexNameRename  map[string]string `config:"index_rename"`
	TypeNameRename   map[string]string `config:"type_rename"`
}

func (filter *ElasticsearchBulkRequestMutate) Name() string {
	return "bulk_request_mutate"
}

func (this *ElasticsearchBulkRequestMutate) Filter(ctx *fasthttp.RequestCtx) {
	contentEncoding := string(ctx.Request.Header.PeekAny([]string{fasthttp.HeaderContentEncoding, fasthttp.HeaderContentEncoding2}))
	pathStr := util.UnsafeBytesToString(ctx.PhantomURI().Path())

	if util.SuffixStr(pathStr, "/_bulk") {
		body := ctx.Request.GetRawBody()

		//this buffer will release after context exit
		var bulkBuff *bytebufferpool.ByteBuffer = bytebufferpool.Get("bulk_mutate_request_docs")
		defer bytebufferpool.Put("bulk_mutate_request_docs", bulkBuff)
		var metaCollected bool
		docCount, err := elastic.WalkBulkRequests(pathStr, body, func(eachLine []byte) (skipNextLine bool) {
			return false
		}, func(metaBytes []byte, actionStr, index, typeName, id, routing string, offset int) (err error) {
			metaCollected = false

			metaStr := util.UnsafeBytesToString(metaBytes)
			var indexNew, typeNew, idNew string

			if (actionStr == elastic.ActionIndex || actionStr == elastic.ActionCreate) && (len(id) == 0 || id == "null") && this.FixNilID {
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

			if typeName == "" && typeNew == "" && !this.RemoveTypeMeta && this.FixNilType && this.DefaultType != "" {
				typeName = this.DefaultType
				typeNew = this.DefaultType
				if global.Env().IsDebug {
					log.Trace("use default type: ", this.DefaultType, " for: ", metaStr)
				}
			}

			//index should not be empty, or will be use the default index
			if index == "" {
				index = this.DefaultIndex
				idNew = this.DefaultIndex
			}

			//handle the index rename
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

			//handle the type rename
			if typeName != "" && !this.RemoveTypeMeta && len(this.TypeNameRename) > 0 {
				v, ok := this.TypeNameRename[typeName]
				if ok && v != typeName {
					typeNew = v
					typeName = v
				} else {
					v, ok := this.TypeNameRename["*"]
					if ok && v != typeName {
						typeNew = v
						typeName = v
					}
				}
			}

			set := map[string]string{}
			remove := map[string]string{}

			if this.RemoveTypeMeta {
				remove["_type"] = "_type"
			} else {
				if typeNew != "" {
					set["_type"] = typeNew
				}
			}

			if this.Pipeline != "" {
				set["pipeline"] = this.Pipeline
			} else if this.RemovePipeline {
				remove["pipeline"] = "pipeline"
			}

			if indexNew != "" {
				set["_index"] = indexNew
			}

			if idNew != "" {
				set["_id"] = idNew
			}

			if len(set) > 0 || len(remove) > 0 {
				metaBytes, err = batchUpdateJson(metaBytes, actionStr, set, remove)
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
				log.Tracef("metadata:\n%v", string(metaBytes))
			}

			elastic.SafetyAddNewlineBetweenData(bulkBuff, metaBytes)
			metaCollected = true

			return nil
		}, func(payloadBytes []byte, actionStr, index, typeName, id, routing string) {

			if metaCollected {
				if global.Env().IsDebug {
					log.Tracef("payload:\n%v", string(payloadBytes))
				}

				if payloadBytes != nil && len(payloadBytes) > 0 {
					elastic.SafetyAddNewlineBetweenData(bulkBuff, payloadBytes)
				}
			}
		}, nil)

		if err != nil {
			log.Errorf("processing: %v docs, err: %v", docCount, err)
			return
		}

		if bulkBuff.Len() > 0 {
			if !util.BytesHasSuffix(bulkBuff.B, elastic.NEWLINEBYTES) {
				bulkBuff.Write(elastic.NEWLINEBYTES)
			}
			if contentEncoding == "gzip" {
				ctx.Request.ResetBody()
				_, err := fasthttp.WriteGzipLevel(ctx.Request.BodyWriter(), bytes.Copy(bulkBuff.Bytes()), fasthttp.CompressBestCompression)
				if err != nil {
					panic(errors.Errorf("failed to compress message: %v", err))
				}
			} else {
				ctx.Request.SetRawBody(bulkBuff.Bytes())
			}
		}
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("bulk_request_mutate", NewElasticsearchBulkRequestMutateFilter, &ElasticsearchBulkRequestMutate{})
}

func NewElasticsearchBulkRequestMutateFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchBulkRequestMutate{
		FixNilID: true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
