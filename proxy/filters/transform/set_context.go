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

/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"fmt"
	"io"

	log "github.com/cihub/seelog"
	"github.com/valyala/fasttemplate"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type SetContext struct {
	ContextKeyword  string                 `config:"context_keyword"`
	VariableKeyword string                 `config:"variable_keyword"`
	Context         map[string]interface{} `config:"context"`
	keys            map[string]interface{}
	valueInContext  bool
	templates       map[string]*fasttemplate.Template
}

func (filter *SetContext) Name() string {
	return "set_context"
}

func (filter *SetContext) Filter(ctx *fasthttp.RequestCtx) {
	var err error
	if filter.keys != nil && len(filter.keys) > 0 {
		for k, v := range filter.keys {
			if filter.valueInContext {
				str, ok := v.(string)
				if ok {

					if filter.templates != nil {
						t, ok := filter.templates[str]
						if ok {
							if t != nil {
								str = t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
									variable, err := ctx.GetValue(tag)
									if err==nil{
										return w.Write([]byte(util.ToString(variable)))
									}
									return -1, err
								})
							}
						}
					}

					//is in context
					if util.ContainStr(str, filter.ContextKeyword) {
						v, err = ctx.GetValue(str)
					} else {
						v = str
					}
				}
			}

			_, err = ctx.PutValue(k, v)
			if err != nil {
				log.Error("key:", k, ",value:", v, ",err:", err)
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("set_context", NewSetContext, &SetContext{})
}

func NewSetContext(c *config.Config) (pipeline.Filter, error) {

	runner := SetContext{
		VariableKeyword: "$[[",
		ContextKeyword:  "_ctx.",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if len(runner.Context) > 0 {
		runner.templates = map[string]*fasttemplate.Template{}
		runner.keys = util.Flatten(runner.Context, false)
		for _, v := range runner.keys {
			str1, ok := v.(string)
			if ok {
				if util.ContainStr(str1, runner.VariableKeyword) {
					runner.valueInContext = true
					template, err := fasttemplate.NewTemplate(str1, runner.VariableKeyword, "]]")
					if err != nil {
						panic(err)
					}
					runner.templates[str1] = template
				}
			}

		}
	}

	return &runner, nil
}
