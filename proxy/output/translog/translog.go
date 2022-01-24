package translog

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/lib/fasthttp"
	"os"
	"path"
)

const SplitLine = "#\r\n\r\n#"

var splitBytes = []byte(SplitLine)

type TranslogOutput struct {
	Path         string              `config:"path"`
	Category     string              `config:"category"`
	Filename     string              `config:"filename"`
	RotateConfig rotate.RotateConfig `config:"rotate"`
}

func (filter *TranslogOutput) Name() string {
	return "translog"
}

func (filter *TranslogOutput) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("saving request to translog")
	}

	logPath := path.Join(filter.Path, "translog", filter.Category)

	os.MkdirAll(logPath, 0755)
	logPath = path.Join(logPath, filter.Filename)

	handler := rotate.GetFileHandler(logPath, filter.RotateConfig)

	data := ctx.Request.Encode()
	_, err := handler.WriteBytesArray(data, splitBytes)
	if err != nil {
		log.Error(err)
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("translog",NewTranslogOutput,&TranslogOutput{})
}

func NewTranslogOutput(c *config.Config) (pipeline.Filter, error) {

	runner := TranslogOutput{
		Path:         global.Env().GetDataDir(),
		Category:     "default",
		Filename:     "translog.log",
		RotateConfig: rotate.DefaultConfig,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
