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

package transform

import (
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/lib/fasttemplate"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type RequestTemplate struct {
	Method      string                 `config:"method"`
	RequestBody string                 `config:"body"`
	Parameters  map[string]interface{} `config:"variable"`
}

type ElasticsearchLookup struct {
	Elasticsearch string          `config:"target.elasticsearch"`
	IndexPattern  string          `config:"target.index_pattern"`
	Template      RequestTemplate `config:"target.template"`

	JoinBySourceFieldValuesKeyPath []string `config:"source.join_by_field_values.json_path"`
}

func (filter *ElasticsearchLookup) Name() string {
	return "elasticsearch_lookup"
}

func (filter *ElasticsearchLookup) Filter(ctx *fasthttp.RequestCtx) {
	body := ctx.Response.GetRawBody()

	if body != nil && len(body) > 0 {

		var joinKeys = []interface{}{}
		var oldDocs = map[interface{}][]byte{}
		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

			//save old docs
			//get key field value as key, `category`
			vData, vType, _, err := jsonparser.Get(value, filter.JoinBySourceFieldValuesKeyPath...)
			if err != nil {
				log.Error(err)
				return
			}

			if vData != nil && len(vData) > 0 {
				var data interface{}
				switch vType {
				case jsonparser.String:
					data = string(vData)
					break
				case jsonparser.Number:
					data, err = jsonparser.ParseInt(vData)
					if err != nil {
						data, err = jsonparser.ParseFloat(vData)
					}
					break
				}

				joinKeys = append(joinKeys, data)
				oldDocs[data] = vData
			}
			//set doc as value
			//collect joinKeys for further query

		}, "hits", "hits")

		if global.Env().IsDebug {
			log.Debug("doc joinKeys:", joinKeys)
		}

		if len(joinKeys) > 0 {
			docs := filter.Lookup(joinKeys)
			if global.Env().IsDebug {
				log.Debug("fetch keys:", joinKeys, ",hit count of docs:", len(docs))
			}

			if len(docs) > 0 {
				offset := -1
				for key, doc := range oldDocs {
					offset++
					newDoc, ok := docs[key]
					if ok {
						body, _ = jsonparser.Set(body, newDoc, "hits", "hits", fmt.Sprintf("[%v]", offset))
						if global.Env().IsDebug {
							log.Trace("update doc from:", string(util.EscapeNewLine(doc)), " => to: ", string(util.EscapeNewLine(newDoc)))
						}
					}
				}
				ctx.Response.SetRawBody(body)
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("elasticsearch_lookup", NewElasticsearchLookupFilter, &ElasticsearchLookup{})
}

func NewElasticsearchLookupFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchLookup{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if len(runner.JoinBySourceFieldValuesKeyPath) == 0 {
		panic(errors.New("join_by_field_values.json_path can't be nil"))
	}

	runner.Template.RequestBody = fasttemplate.ExecuteStringStd(runner.Template.RequestBody, "{{", "}}", runner.Template.Parameters)

	return &runner, nil
}

func (filter *ElasticsearchLookup) Lookup(docsID []interface{}) map[interface{}][]byte {
	para := map[string]interface{}{
		"JOIN_BY_FIELD_ARRAY_VALUES": string(util.JoinInterfaceArray(docsID, ",", func(str string) string {
			return "\"" + str + "\""
		})),
		"MAX_NUMBER_OF_FIELD_ARRAY_VALUES": strconv.Itoa(len(docsID)),
	}

	dsl := util.UnsafeStringToBytes(fasttemplate.ExecuteString(filter.Template.RequestBody, "{{", "}}", para))

	if global.Env().IsDebug {
		log.Debug(filter.Template.RequestBody)
		log.Trace(string(dsl))
	}

	if global.Env().IsDebug {
		log.Debug("index:", filter.IndexPattern, ", dsl:", string(dsl))
	}

	res, err := elastic.GetClient(filter.Elasticsearch).SearchWithRawQueryDSL(filter.IndexPattern, dsl)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	if res != nil && res.RawResult != nil && res.RawResult.Body != nil && len(res.RawResult.Body) > 0 {
		result := map[interface{}][]byte{}
		jsonparser.ArrayEach(res.RawResult.Body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			keyData, keyType, _, err := jsonparser.Get(value, "key")
			var key interface{}
			switch keyType {
			case jsonparser.String:
				key = string(keyData)
				break
			case jsonparser.Number:
				key, err = jsonparser.ParseInt(keyData)
				if err != nil {
					key, err = jsonparser.ParseFloat(keyData)
				}
				break
			}

			docCount, err := jsonparser.GetInt(value, "doc_count")
			if docCount > 1 {
				jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					result[key] = value
				}, "sorted_doc", "hits", "hits")
				if err != nil {
					log.Error(err)
					panic(err)
				}
			}

		}, "aggregations", "hit_docs", "buckets")

		if len(result) > 0 {
			return result
		}
	}

	return nil
}
