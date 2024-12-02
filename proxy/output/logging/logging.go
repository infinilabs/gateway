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

package logging

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fastjson_marshal"
	"infini.sh/gateway/common/model"

	"time"

	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
)

type RequestLogging struct {
	config *Config
}

type Config struct {
	MinElaspsedTimeInMs int    `config:"min_elapsed_time_in_ms"`
	MaxRequestBodySize  int    `config:"max_request_body_size"`
	MaxResponseBodySize int    `config:"max_response_body_size"`
	SaveBulkDetails     bool   `config:"bulk_stats_details"`
	FormatHeaderKey     bool   `config:"format_header_keys"`
	RemoveAuthHeaderKey bool   `config:"remove_authorization"`
	QueueName           string `config:"queue_name"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("logging", New, &Config{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	cfg := Config{
		MinElaspsedTimeInMs: -1,
		MaxRequestBodySize:  1024,
		MaxResponseBodySize: 1024,
		SaveBulkDetails:     true,
		FormatHeaderKey:     false,
		RemoveAuthHeaderKey: true,
		QueueName:           "logging",
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := RequestLogging{config: &cfg}
	initPool()

	return &runner, nil
}

func (this *RequestLogging) Name() string {
	return "logging"
}

var writerPool *sync.Pool
var requstObjPool *sync.Pool
var reqPool *sync.Pool
var resPool *sync.Pool
var reqFlowPool *sync.Pool

func initPool() {
	if writerPool != nil {
		return
	}
	writerPool = &sync.Pool{
		New: func() interface{} {
			return new(fastjson_marshal.Writer)
		},
	}

	requstObjPool = &sync.Pool{
		New: func() interface{} {
			return new(model.HttpRequest)
		},
	}
	reqPool = &sync.Pool{
		New: func() interface{} {
			return new(model.Request)
		},
	}
	resPool = &sync.Pool{
		New: func() interface{} {
			return new(model.Response)
		},
	}
	reqFlowPool = &sync.Pool{
		New: func() interface{} {
			return new(model.DataFlow)
		},
	}
}

func (this *RequestLogging) Filter(ctx *fasthttp.RequestCtx) {
	elapsedTimeInMs := float32(float64(ctx.GetElapsedTime().Microseconds()) * 0.001)

	if this.config.MinElaspsedTimeInMs > 0 {
		if this.config.MinElaspsedTimeInMs >= int(elapsedTimeInMs) {
			ctx.Finished()
			return
		}
	}

	request := requstObjPool.Get().(*model.HttpRequest)
	request.Request = reqPool.Get().(*model.Request)
	request.Response = resPool.Get().(*model.Response)
	request.DataFlow = reqFlowPool.Get().(*model.DataFlow)

	defer requstObjPool.Put(request)
	defer resPool.Put(request.Response)
	defer reqPool.Put(request.Request)
	defer reqFlowPool.Put(request.DataFlow)

	request.LoggingTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	request.Request.StartTime = ctx.Time().UTC().Format("2006-01-02T15:04:05.000Z")

	request.IsTLS = ctx.IsTLS()
	if ctx.IsTLS() {
		request.TLSDidResume = ctx.TLSConnectionState().DidResume
	}

	request.Request.Method = string(ctx.Method())
	request.Request.URI = ctx.PhantomURI().String()
	request.Request.Path = string(ctx.Path())

	m := map[string]string{}
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		m[string(key)] = string(value)
	})

	request.Request.QueryArgs = m

	request.Request.Host = string(ctx.Request.Host())

	if ctx.LocalIP() != nil {
		request.LocalIP = ctx.LocalIP().String()
	}
	if ctx.RemoteIP() != nil {
		request.RemoteIP = ctx.RemoteIP().String()
	}

	if ctx.RemoteAddr() != nil {
		request.Request.RemoteAddr = ctx.RemoteAddr().String()
	}

	if ctx.LocalAddr() != nil {
		request.Request.LocalAddr = ctx.LocalAddr().String()
	}

	reqBody := util.UnsafeBytesToString(ctx.Request.GetRawBody())

	if len(reqBody) > this.config.MaxRequestBodySize {
		reqBody = reqBody[0:this.config.MaxRequestBodySize]
	}

	request.Request.Body = reqBody
	request.Request.BodyLength = ctx.Request.GetBodyLength()

	request.Response.ElapsedTimeInMs = elapsedTimeInMs

	if ctx.Response.LocalAddr() != nil {
		request.Response.LocalAddr = ctx.Response.LocalAddr().String()
	} else {
		request.Response.LocalAddr = ""
	}

	request.Elastic = map[string]interface{}{}

	if this.config.SaveBulkDetails {

		if ctx.Has("bulk_response_status") {
			bulk_status := ctx.Get("bulk_response_status")
			if bulk_status != nil {
				request.Elastic["bulk_results"] = bulk_status
			}
		}

		if ctx.Has("bulk_index_stats") {
			bulk_status := map[string]interface{}{}
			if request.Elastic == nil {
				request.Elastic = map[string]interface{}{}
			}
			indexStats := ctx.Get("bulk_index_stats")
			if indexStats != nil {
				statsObj, ok := indexStats.(map[string]int)
				if ok {
					docs := 0
					for _, v := range statsObj {
						docs += v
					}

					bulk_status["indices"] = len(statsObj)
					bulk_status["documents"] = docs

					actionStats := ctx.Get("bulk_action_stats")
					stats := map[string]interface{}{}
					stats["index"] = indexStats
					stats["action"] = actionStats
					bulk_status["stats"] = stats

					request.Elastic["bulk_requests"] = bulk_status
				}
			}
		}
	}

	if ctx.Has("elastic_cluster_name") {
		stats := ctx.Get("elastic_cluster_name")
		if stats != nil {
			request.Elastic["cluster_name"] = stats
		}
	}

	//request.DataFlow = &model.DataFlow{}
	request.DataFlow.From = request.RemoteIP

	process := ctx.GetRequestProcess()
	if len(ctx.GetRequestProcess()) > 0 {
		request.DataFlow.Process = strings.Split(process, "->")
	}

	//TODO ,use gateway's uuid instead
	request.DataFlow.Relay = global.Env().SystemConfig.NodeConfig.ToString()

	if len(ctx.Destination()) > 0 {
		request.DataFlow.To = ctx.Destination()
	} else if ctx.Response.RemoteAddr() != nil {
		request.Response.RemoteAddr = ctx.Response.RemoteAddr().String()
		request.DataFlow.To = []string{request.Response.RemoteAddr}
	}

	request.Response.Cached = ctx.Response.Cached
	request.Response.StatusCode = ctx.Response.StatusCode()

	respBody := string(ctx.Response.GetRawBody())

	if global.Env().IsDebug {
		log.Trace("logging request body:", string(reqBody))
		log.Trace("logging response body:", string(respBody))
	}

	if len(respBody) > this.config.MaxResponseBodySize {
		respBody = respBody[0:this.config.MaxResponseBodySize]
	}

	request.Response.Body = respBody

	request.Response.BodyLength = ctx.Response.GetBodyLength()

	//parser user
	exists, user, _ := ctx.Request.ParseBasicAuth()
	if exists {
		request.Request.User = string(user)
	}

	m = map[string]string{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {

		tempKey := string(key)
		if this.config.RemoveAuthHeaderKey {
			if util.ContainsAnyInArray(tempKey, fasthttp.AuthHeaderKeys) {
				return
			}
		}
		if this.config.FormatHeaderKey {
			m[strings.ToLower(tempKey)] = string(value)
		} else {
			m[tempKey] = string(value)
		}
	})

	request.Request.Header = m

	m = map[string]string{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		if this.config.FormatHeaderKey {
			m[strings.ToLower(string(key))] = string(value)
		} else {
			m[string(key)] = string(value)
		}
	})

	request.Response.Header = m

	bytes, err := request.MarshalJSON()
	if err != nil {
		panic(err)
	}

	err = queue.Push(queue.GetOrInitConfig(this.config.QueueName), bytes)
	if err != nil {
		panic(err)
	}

}
