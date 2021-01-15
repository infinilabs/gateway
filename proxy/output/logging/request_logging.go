package logging

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fastjson_marshal"
	"infini.sh/gateway/common/model"
	"strings"
	"sync"

	"infini.sh/framework/core/queue"
	//jsoniter "github.com/json-iterator/go"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type RequestLogging struct {
	param.Parameters
}

func (this RequestLogging) Name() string {
	return "request_logging"
}

//var lock sync.Mutex
var writerPool *sync.Pool
var requstObjPool *sync.Pool
var reqPool *sync.Pool
var resPool *sync.Pool

func initPool() {
	if writerPool != nil {
		return
	}
	writerPool = &sync.Pool{
		New: func() interface{} {
			return new(fastjson_marshal.Writer)
		},
	}

	requstObjPool = &sync.Pool {
		New: func() interface{} {
			return new(model.HttpRequest)
		},
	}
	reqPool = &sync.Pool {
		New: func() interface{} {
			return new(model.Request)
		},
	}
	resPool = &sync.Pool {
		New: func() interface{} {
			return new(model.Response)
		},
	}
}

func (this RequestLogging) Process(ctx *fasthttp.RequestCtx) {

	initPool()

	if global.Env().IsDebug {
		log.Trace("start logging requests")
	}

	request := requstObjPool.Get().(*model.HttpRequest)
	request.Request = reqPool.Get().(*model.Request)
	request.Response = resPool.Get().(*model.Response)

	defer requstObjPool.Put(request)
	defer resPool.Put(request.Response)
	defer reqPool.Put(request.Request)

	request.ID = ctx.ID()
	request.ConnTime = ctx.ConnTime().UTC().Format("2006-01-02T15:04:05.000Z")
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

	ce := string(ctx.Request.Header.Peek(fasthttp.HeaderContentEncoding))
	if ce == "gzip" {
		body, err := ctx.Request.BodyGunzip()
		if err != nil {
			panic(err)
		}
		request.Request.Body = string(body)
	} else if ce == "deflate" {
		body, err := ctx.Request.BodyInflate()
		if err != nil {
			panic(err)
		}
		request.Request.Body = string(body)
	} else {
		request.Request.Body = string(ctx.Request.Body())
	}

	request.Request.BodyLength = ctx.Request.GetBodyLength()

	request.Response.ElapsedTimeInMs = float32(float64(ctx.GetElapsedTime().Microseconds()) * 0.001)

	if ctx.Response.LocalAddr() != nil {
		request.Response.LocalAddr = ctx.Response.LocalAddr().String()
	}

	request.DataFlow = &model.DataFlow{}
	request.DataFlow.From = request.RemoteIP

	//TODO ,use gateway's uuid instead
	request.DataFlow.Relay = request.Request.LocalAddr

	if len(ctx.Response.Destination()) > 0 {
		request.DataFlow.To = ctx.Response.Destination()
	} else if ctx.Response.RemoteAddr() != nil {
		request.Response.RemoteAddr = ctx.Response.RemoteAddr().String()
		request.DataFlow.To = []string{request.Response.RemoteAddr}
	}

	request.Response.Cached = ctx.Response.Cached
	request.Response.StatusCode = ctx.Response.StatusCode()

	ce = string(ctx.Response.Header.Peek(fasthttp.HeaderContentEncoding))
	if ce == "" {
		ce = string(ctx.Response.Header.Peek("content-encoding"))
	}
	if ce == "gzip" {
		body, err := ctx.Response.BodyGunzip()
		if err != nil {
			panic(err)
		}
		request.Response.Body = string(body)
	} else if ce == "deflate" {
		body, err := ctx.Response.BodyInflate()
		if err != nil {
			panic(err)
		}
		request.Response.Body = string(body)
	} else {
		request.Response.Body = string(ctx.Response.Body())
	}

	request.Response.BodyLength = len(request.Response.Body)

	formatHeaderKey:=this.GetBool("format_header_keys",false)
	removeAuthHeaderKey:=this.GetBool("remove_authorization",true)

	m = map[string]string{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {

		tempKey:=string(key)
		if removeAuthHeaderKey{
			if util.ContainsAnyInArray(tempKey,fasthttp.AuthHeaderKeys){
				return
			}
		}
		if formatHeaderKey{
			m[strings.ToLower(tempKey)] = string(value)
		}else{
			m[tempKey] = string(value)
		}
	})

	if len(m) > 0 {
		request.Request.Header = m
	}

	exists, user, _ := ctx.ParseBasicAuth()
	if exists {
		request.Request.User = string(user)
	}

	m = map[string]string{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		if formatHeaderKey {
			m[strings.ToLower(string(key))] = string(value)
		}else{
			m[string(key)] = string(value)
		}
	})

	if len(m) > 0 {
		request.Response.Header = m
	}

	//lock.Lock()
	var w *fastjson_marshal.Writer
	v := writerPool.Get()
	if v != nil {
		w = v.(*fastjson_marshal.Writer)
		w.Reset()
	}

	defer writerPool.Put(w)

	err := request.MarshalFastJSON(w)
	if err != nil {
		panic(err)
	}

	////verify json
	//if false {
	//	data := w.Bytes()
	//	v := model.HttpRequest{}
	//	util.FromJSONBytes(data, &v)
	//}

	//var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//bytes, err := json.Marshal(&request)
	//if err != nil {
	//	panic(err)
	//}

	//fmt.Println("logging now", string(w.Bytes()))

	err = queue.Push(this.GetStringOrDefault("queue_name", "request_logging"), w.Bytes())
	//lock.Unlock()
	if err != nil {
		panic(err)
	}

}
