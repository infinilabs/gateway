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

/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package echo

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasttemplate"
	"io"
	"strings"
)

type Echo struct {
	RepeatTimes int  `config:"repeat"    type:"number"  default_value:"1" `
	Status      int  `config:"status"    type:"status"  default_value:"200" `
	Continue    bool `config:"continue"  type:"bool"    default_value:"true" `
	Terminal    bool `config:"stdout"    type:"bool"    default_value:"false" `

	Logging      bool     `config:"logging"    type:"bool"    default_value:"false" `
	LoggingLevel string   `config:"logging_level"    type:"string"    default_value:"info" `
	Response     bool     `config:"response"    type:"bool"    default_value:"true" `
	Message      string   `config:"message"   type:"string"  default_value:"." `
	Messages     []string `config:"messages" `
	template     *fasttemplate.Template
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("echo", New, &Echo{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	runner := Echo{
		Response:     true,
		Status:       200,
		Logging:      false,
		Terminal:     false,
		RepeatTimes:  1,
		Continue:     true,
		LoggingLevel: "info",
		Message:      ".",
	}
	var err error

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if strings.Contains(runner.Message, "$[[") {
		runner.template, err = fasttemplate.NewTemplate(runner.Message, "$[[", "]]")
		if err != nil {
			panic(err)
		}
	}

	return &runner, nil
}

func (filter *Echo) Name() string {
	return "echo"
}

func (filter *Echo) Filter(ctx *fasthttp.RequestCtx) {
	str := filter.Message

	if filter.template != nil {
		str = filter.template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			variable, err := ctx.GetValue(tag)
			if err == nil {
				return w.Write([]byte(util.ToString(variable)))
			}
			return -1, err
		})
	}

	size := filter.RepeatTimes
	for i := 0; i < size; i++ {
		if filter.Response {
			ctx.WriteString(str)
		}
		if filter.Terminal {
			fmt.Print(str)
		}
		if filter.Logging {
			switch filter.LoggingLevel {
			case "info":
				log.Info(str)
				break
			case "debug":
				log.Debug(str)
				break
			case "warn":
				log.Warn(str)
				break
			case "error":
				log.Error(str)
				break
			}
		}
	}

	if filter.Response {
		ctx.Response.SetStatusCode(filter.Status)
	}

	if !filter.Continue {
		ctx.Finished()
	}
}
