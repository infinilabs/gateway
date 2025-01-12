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
	pipeline.RegisterFilterPluginWithConfigMetadata("translog", NewTranslogOutput, &TranslogOutput{})
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
