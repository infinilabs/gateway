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
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"path"
)

type RequestRecord struct {
	QueueName string `config:"queue_name"`
	FileName  string `config:"filename"`
	Verbose   bool   `config:"stdout"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("record", NewRequestRecord, &Config{})
}

func NewRequestRecord(c *config.Config) (pipeline.Filter, error) {

	runner := RequestRecord{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (this *RequestRecord) Name() string {
	return "record"
}

const tab = "  "
const newline = "\n"
const args = "?"

func (this *RequestRecord) Filter(ctx *fasthttp.RequestCtx) {

	buffer := bytebufferpool.Get("record")
	defer bytebufferpool.Put("record", buffer)

	buffer.Write(ctx.Method())
	buffer.WriteString(tab)
	buffer.Write(ctx.Path())
	if ctx.QueryArgs() != nil {
		argsStr := ctx.QueryArgs().QueryString()
		if len(argsStr) > 0 {
			buffer.WriteString(args)
			buffer.Write(argsStr)
		}

	}
	buffer.WriteString(newline)

	reqBody := ctx.Request.GetRawBody()
	if len(reqBody) > 0 {
		buffer.Write(reqBody)
		buffer.WriteString(newline)
	}

	req := buffer.String()
	if this.FileName != "" {
		util.FileAppendNewLine(path.Join(global.Env().GetDataDir(), this.FileName), req)
	}
	if this.Verbose {
		fmt.Println(req)
	}

}
