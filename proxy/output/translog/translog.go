package translog

import (
	"bufio"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"os"
	"path"
	"sync"
)

var f *os.File
var err error

const defaultBufSize = 8192

var w *bufio.Writer

func Open() {
	logPath:=path.Join(global.Env().GetWorkingDir(),"translog/","default/")
	os.MkdirAll(logPath,0755)
	//TODO rotate log files
	file:=path.Join(logPath,"1.log")
	f, err = os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	w = bufio.NewWriterSize(f, defaultBufSize)
	lock = &sync.Mutex{}
}

func Close() {
	if w!=nil{
		err = w.Flush()
		if err != nil {
			panic(err)
		}
		err = f.Close()
		if err != nil {
			panic(err)
		}
	}
}

func Flush() {
	err = w.Flush()
	if err != nil {
		panic(err)
	}
}

func Sync() {
	err = f.Sync()
	if err != nil {
		panic(err)
	}
}

//var batch = 1000
//var hit = 0
//
//var msg *capnp.Message
//var docs cap.Request_List
//
//func initBatch() {
//
//	a := capnp.MultiSegment(nil)
//	var seg *capnp.Segment
//	msg, seg, err = capnp.NewMessage(a)
//	if err != nil {
//		panic(err)
//	}
//
//	root, err := cap.NewRootRequestGroup(seg)
//	if err != nil {
//		panic(err)
//	}
//
//	docs, err = root.NewRequests(int32(batch))
//	if err != nil {
//		panic(err)
//	}
//}
//


func SaveRequest(ctx *fasthttp.RequestCtx) {
	lock.Lock()

	data:=ctx.Request.Encode()
	bufWriteContent(&data)

	//if hit == 0 {
	//	initBatch()
	//}
	//
	//if hit < batch {
	//	d := docs.At(hit)
	//	err = d.SetMethod(ctx.Method())
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	err = d.SetUrl(ctx.Request.URI().RequestURI())
	//	if err != nil {
	//		panic(err)
	//	}
	//	err = d.SetBody(ctx.Request.Body())
	//	if err != nil {
	//		panic(err)
	//	}
	//	hit++
	//} else {
	//	d, err := msg.Marshal()
	//	if err != nil {
	//		panic(err)
	//	}
	//	bufWriteContent(&d)
	//	hit = 0
	//
	//}

	lock.Unlock()
}

//func writeMmap(data *[]byte)  {
//	_, err = f.Write(*data)
//	if err != nil {
//		panic(err)
//	}
//
//	_, err= f.Write([]byte("\n"))
//	if err != nil {
//		panic(err)
//	}
//}
//
//func gzipWrite(data *[]byte)  {
//	lock.Lock()
//
//	if _, err := gz.Write(*data); err != nil {
//		panic(err)
//	}
//	_, err= w.Write([]byte("\n"))
//	if err != nil {
//		panic(err)
//	}
//
//	lock.Unlock()
//}

const SplitLine = "#\r\n\r\n#"

var splitBytes = []byte(SplitLine)

var lock *sync.Mutex

func bufWriteContent(data *[]byte) {
	//fmt.Println(*data)
	if _, err = w.Write(*data); err != nil {
		panic(err)
	}
	_, err = w.Write(splitBytes)
	if err != nil {
		panic(err)
	}
	err=w.Flush()
	if err != nil {
		panic(err)
	}
}




//var jsonOK = "{ \"took\" : 1, \"errors\" : false }"
//var bulkRequestOKBody = []byte(jsonOK)


type TranslogOutput struct {
	param.Parameters
}

func (filter TranslogOutput) Name() string {
	return "translog"
}

func (filter TranslogOutput) Process(ctx *fasthttp.RequestCtx) {

	//if p.proxyConfig.AsyncWrite && strings.Contains(ctx.URI().String(), "_bulk") {


		//stats.Increment("request", "action.bulk")

		if global.Env().IsDebug {
			log.Trace("saving bulk request")
		}

		SaveRequest(ctx)

		//
		//ctx.Response.SetStatusCode(http.StatusOK)
		//ctx.Response.SetBody(bulkRequestOKBody)

	//}

}



