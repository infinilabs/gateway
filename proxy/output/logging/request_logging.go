package logging

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
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

func initPool() {
	if writerPool!=nil{
		return
	}
	writerPool = &sync.Pool {
		New: func()interface{} {
			return new(fastjson_marshal.Writer)
		},
	}
}

func (this RequestLogging) Process(ctx *fasthttp.RequestCtx) {

	initPool()

	if global.Env().IsDebug {
		log.Trace("start logging requests")
	}

	request := model.HttpRequest{}
	request.Request = &model.Request{}
	request.Response = &model.Response{}

	request.ID = ctx.ID()
	request.ConnTime = ctx.ConnTime().UTC().Format("2006-01-02T15:04:05.000Z")
	request.LoggingTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	request.Request.StartTime = ctx.Time().UTC().Format("2006-01-02T15:04:05.000Z")

	request.IsTLS = ctx.IsTLS()
	if ctx.IsTLS(){
		request.TLSDidResume = ctx.TLSConnectionState().DidResume
	}

	request.Request.Method = string(ctx.Method())
	request.Request.URI = ctx.URI().String()
	request.Request.Path = string(ctx.Path())

	request.Request.QueryArgs = map[string]string{}
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		request.Request.QueryArgs[string(key)] = string(value)
	})

	request.Request.Host = string(ctx.Request.Host())

	request.LocalIP = ctx.LocalIP().String()
	request.RemoteIP = ctx.RemoteIP().String()

	request.Request.RemoteAddr =ctx.RemoteAddr().String()
	request.Request.LocalAddr =ctx.LocalAddr().String()


	ce := string(ctx.Request.Header.Peek(fasthttp.HeaderContentEncoding))
	if ce == "gzip" {
		body,err:=ctx.Request.BodyGunzip()
		if err!=nil{
			panic(err)
		}
		request.Request.Body = string(body)
	}else if ce=="deflate"{
		body,err:=ctx.Request.BodyInflate()
		if err!=nil{
			panic(err)
		}
		request.Request.Body = string(body)
	}else{
		request.Request.Body = string(ctx.Request.Body())
	}

	request.Request.BodyLength = len(request.Request.Body)

	request.Response.ElapsedTimeInMs = float32(float64(ctx.GetElapsedTime().Microseconds()) *0.001)

	if ctx.Response.LocalAddr() != nil {
		request.Response.LocalAddr = ctx.Response.LocalAddr().String()
	}

	if ctx.Response.RemoteAddr() != nil {
		request.Response.RemoteAddr = ctx.Response.RemoteAddr().String()
	}

	request.Response.Cached = ctx.Response.Cached
	request.Response.StatusCode = ctx.Response.StatusCode()

	request.DataFlow=&model.DataFlow{}
	request.DataFlow.From=request.RemoteIP
	request.DataFlow.Relay=request.Request.LocalAddr
	request.DataFlow.To=request.Response.RemoteAddr

	if request.Response.Cached && request.Response.RemoteAddr==""{
		request.DataFlow.To="cache"
	}

	ce = string(ctx.Response.Header.Peek(fasthttp.HeaderContentEncoding))
	if ce ==""{
		ce = string(ctx.Response.Header.Peek("content-encoding"))
	}
	if ce == "gzip" {
		body,err:=ctx.Response.BodyGunzip()
		if err!=nil{
			panic(err)
		}
		request.Response.Body = string(body)
	}else if ce=="deflate"{
		body,err:=ctx.Response.BodyInflate()
		if err!=nil{
			panic(err)
		}
		request.Response.Body = string(body)
	}else{
		request.Response.Body = string(ctx.Response.Body())
	}

	request.Response.BodyLength = len(request.Response.Body)

	request.Request.Header = map[string]string{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {

		//TODO remove duplicated headers

		//TODO header may need to keep original case, no need to lowercase
		request.Request.Header[strings.ToLower(string(key))] = string(value)
	})

	exists,user,_:=ctx.ParseBasicAuth()
	if exists{
		request.Request.User=string(user)
	}


	request.Response.Header = map[string]string{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		//TODO header may need to keep original case, no need to lowercase
		request.Response.Header[strings.ToLower(string(key))] = string(value)
	})

	//lock.Lock()
	var w *fastjson_marshal.Writer
	v:=writerPool.Get()
	if v!=nil{
		w=v.(*fastjson_marshal.Writer)
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

	err = queue.Push(this.GetStringOrDefault("queue_name","request_logging"),w.Bytes() )
	//lock.Unlock()
	if err != nil {
		panic(err)
	}

}
