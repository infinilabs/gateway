package logging

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fastjson_marshal"
	"infini.sh/gateway/common/model"
	"strings"
	"sync"

	"infini.sh/framework/core/queue"
	"infini.sh/framework/lib/fasthttp"
	"time"
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

	request := requstObjPool.Get().(*model.HttpRequest)
	request.Request = reqPool.Get().(*model.Request)
	request.Response = resPool.Get().(*model.Response)
	request.DataFlow = reqFlowPool.Get().(*model.DataFlow)

	defer requstObjPool.Put(request)
	defer resPool.Put(request.Response)
	defer reqPool.Put(request.Request)
	defer reqFlowPool.Put(request.DataFlow)

	if this.config.MinElaspsedTimeInMs > 0 {
		if this.config.MinElaspsedTimeInMs >= int(request.Response.ElapsedTimeInMs) {
			ctx.Finished()
		}
	}

	//request.ID = ctx.ID()

	request.ID = uint64(ctx.SequenceID)

	//request.ConnTime = ctx.ConnTime().UTC().Format("2006-01-02T15:04:05.000Z")
	request.LoggingTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	request.Request.StartTime = ctx.Time().UTC().Format("2006-01-02T15:04:05.000Z")

	request.IsTLS = ctx.IsTLS()
	if ctx.IsTLS() {
		request.TLSDidResume = ctx.TLSConnectionState().DidResume
	}

	request.Request.Method = string(ctx.Method())
	request.Request.URI = ctx.URI().String()
	request.Request.Path = string(ctx.Path())

	m := map[string]string{}
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		m[string(key)] = string(value)
	})

	if len(m) > 0 {
		request.Request.QueryArgs = m
	}

	request.Request.Host = string(ctx.Request.Host())

	request.LocalIP = ctx.LocalIP().String()
	request.RemoteIP = ctx.RemoteIP().String()

	request.Request.RemoteAddr = ctx.RemoteAddr().String()
	request.Request.LocalAddr = ctx.LocalAddr().String()

	reqBody := util.UnsafeBytesToString(ctx.Request.GetRawBody())

	if len(reqBody) > this.config.MaxRequestBodySize {
		reqBody = reqBody[0:this.config.MaxRequestBodySize]
	}

	request.Request.Body = reqBody
	request.Request.BodyLength = ctx.Request.GetBodyLength()

	request.Response.ElapsedTimeInMs = float32(float64(ctx.GetElapsedTime().Microseconds()) * 0.001)

	if ctx.Response.LocalAddr() != nil {
		request.Response.LocalAddr = ctx.Response.LocalAddr().String()
	}

	if ctx.Has("bulk_index_stats") {
		if request.Elastic == nil {
			request.Elastic = map[string]interface{}{}
		}
		indexStats := ctx.Get("bulk_index_stats")
		statsObj, ok := indexStats.(map[string]int)
		if ok {
			docs := 0
			for _, v := range statsObj {
				docs += v
			}

			bulk_status := map[string]interface{}{}
			bulk_status["indices"] = len(statsObj)
			bulk_status["documents"] = docs

			if this.config.SaveBulkDetails {
				actionStats := ctx.Get("bulk_action_stats")
				stats := map[string]interface{}{}
				stats["index"] = indexStats
				stats["action"] = actionStats
				bulk_status["stats"] = stats
			}

			request.Elastic["bulk_stats"] = bulk_status
		}
	}

	if ctx.Has("elastic_cluster_name") {
		if request.Elastic == nil {
			request.Elastic = map[string]interface{}{}
		}
		stats := ctx.Get("elastic_cluster_name")
		request.Elastic["cluster_name"] = stats
	}

	//request.DataFlow = &model.DataFlow{}
	request.DataFlow.From = request.RemoteIP

	request.DataFlow.Process = ctx.GetRequestProcess()

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

	//ce = string(ctx.Response.Header.PeekAny([]string{fasthttp.HeaderContentEncoding,"Content-Encoding"}))

	//log.Error(request.Request.URI,",",ce,",",string(util.EscapeNewLine(ctx.Response.Header.Header())))
	//log.Error(ctx.Response.Header.String())

	respBody := string(ctx.Response.GetRawBody())
	if global.Env().IsDebug {
		log.Debug("response body:", string(respBody))
	}

	if len(respBody) > this.config.MaxResponseBodySize {
		respBody = respBody[0:this.config.MaxResponseBodySize]
	}

	request.Response.Body = respBody

	request.Response.BodyLength = ctx.Response.GetBodyLength()

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

	if len(m) > 0 {
		request.Request.Header = m
	}

	exists, user, _ := ctx.Request.ParseBasicAuth()
	if exists {
		request.Request.User = string(user)
	}

	m = map[string]string{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		if this.config.FormatHeaderKey {
			m[strings.ToLower(string(key))] = string(value)
		} else {
			m[string(key)] = string(value)
		}
	})

	if len(m) > 0 {
		request.Response.Header = m
	}

	bytes, err := request.MarshalJSON()
	if err != nil {
		panic(err)
	}

	err = queue.Push(queue.GetOrInitConfig(this.config.QueueName), bytes)
	if err != nil {
		panic(err)
	}

}
