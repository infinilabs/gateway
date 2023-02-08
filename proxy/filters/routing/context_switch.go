/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type ContextSwitchFilter struct {
	Context            string     `config:"context"`
	ContinueAfterMatch bool       `config:"continue"`
	SkipError          bool       `config:"skip_error"`
	StringifyValue     bool       `config:"stringify_value"`
	DefaultAction      string     `config:"default_action"`
	DefaultFlow        string     `config:"default_flow"`
	Switch             []CaseRule `config:"switch"`

	cases       map[interface{}]CaseRule
	defaultFlow common.FilterFlow
}

type CaseRule struct {
	Case   []interface{} `config:"case"`
	CaseValueType string  `config:"case_value_type"`
	Action string        `config:"action"`
	Flow   string        `config:"flow"`
	flow   common.FilterFlow
}

func (filter *ContextSwitchFilter) Name() string {
	return "context_switch"
}

func (filter *ContextSwitchFilter) Filter(ctx *fasthttp.RequestCtx) {
	if len(filter.Switch) <= 0 {
		return
	}
	if filter.Context != "" {
		key, err := ctx.GetValue(filter.Context)
		if err != nil {
			if filter.SkipError {
				return
			}
			panic(errors.Errorf("context_parse,url:%v,err:%v", ctx.Request.URI().String(), err))
		}

		if len(filter.cases) > 0 {
			if filter.StringifyValue {
				key=util.ToString(key)
			}
			v, ok := filter.cases[key]
			if ok {
				if v.Action == redirectAction {
					if v.Flow != "" {
						v.flow.Process(ctx)
						if !filter.ContinueAfterMatch {
							ctx.Finished()
						}
					}
				} else if v.Action == dropAction {
					ctx.Finished()
				}
			} else {
				if filter.DefaultAction == redirectAction {
					if filter.DefaultFlow != "" {
						filter.defaultFlow.Process(ctx)
						if !filter.ContinueAfterMatch {
							ctx.Finished()
						}
					}
				} else if filter.DefaultAction == dropAction {
					ctx.Finished()
				}
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_switch", NewContextSwitchFlowFilter, &ContextSwitchFilter{})
}

func NewContextSwitchFlowFilter(c *config.Config) (pipeline.Filter, error) {
	var err error
	runner := ContextSwitchFilter{
		DefaultAction:  redirectAction,
		StringifyValue: true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.cases = map[interface{}]CaseRule{}

	for _, v := range runner.Switch {
		if v.Action == "" {
			v.Action = runner.DefaultAction
		}

		if v.Action == redirectAction {
			if v.Flow == "" {
				v.Flow = runner.DefaultFlow
			}
		}

		if v.Flow != "" {
			v.flow, err = common.GetFlow(v.Flow)
			if err != nil {
				panic(err)
			}
		}

		for _, v1 := range v.Case {
			if runner.StringifyValue {
				runner.cases[util.ToString(v1)] = v
			}else{
				if v.CaseValueType=="int"{
					v2:=util.InterfaceToInt(v1)
					runner.cases[v2] = v
				}else{
					runner.cases[v1] = v
				}
			}
		}
	}

	if runner.DefaultFlow != "" {
		runner.defaultFlow, err = common.GetFlow(runner.DefaultFlow)
		if err != nil {
			panic(err)
		}
	}

	return &runner, nil
}
