package translog

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/lib/fasthttp"
	"os"
	"path"
)

const SplitLine = "#\r\n\r\n#"

var splitBytes = []byte(SplitLine)

type TranslogOutput struct {
	param.Parameters
}

func (filter TranslogOutput) Name() string {
	return "translog"
}

func (filter TranslogOutput) Process(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("saving request to translog")
	}

	logPath := path.Join(
		filter.GetStringOrDefault("path",global.Env().GetWorkingDir()), "translog",
		filter.GetStringOrDefault("category","default"))
	os.MkdirAll(logPath, 0755)
	logPath=path.Join(logPath,filter.GetStringOrDefault("filename","translog.log"))

	config:=rotate.RotateConfig{
		Compress:         filter.GetBool("compress",true),
		MaxFileAge:       filter.GetIntOrDefault("max_file_age",0),
		MaxFileCount: filter.GetIntOrDefault("max_file_count",100),
		MaxFileSize:      filter.GetIntOrDefault("max_file_size_in_mb",1024),
	}

	handler:=rotate.GetFileHandler(logPath,config)

	data := ctx.Request.Encode()
	_, err := handler.WriteBytesArray(data,splitBytes)
	if err != nil {
		log.Error(err)
	}

}
