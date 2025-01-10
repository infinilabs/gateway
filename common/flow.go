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

package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"strings"
	"sync"
)

type FilterFlow struct {
	orm.ORMObjectBase
	Filters []pipeline.Filter `json:"filters,omitempty"`
}

func (flow *FilterFlow) JoinFilter(filter pipeline.Filter) *FilterFlow {
	if filter == nil || filter.Filter == nil {
		panic("invalid filer")
	}
	flow.Filters = append(flow.Filters, filter)
	return flow
}

func (flow *FilterFlow) ToString() string {
	str := strings.Builder{}
	has := false
	for _, v := range flow.Filters {
		if has {
			str.WriteString(" > ")
		}
		str.WriteString(v.Name())
		has = true
	}
	return str.String()
}

func (flow *FilterFlow) Process(ctx *fasthttp.RequestCtx) {
	for _, v := range flow.Filters {
		if v == nil {
			panic("invalid filter")
		}
		if !ctx.ShouldContinue() {
			if global.Env().IsDebug {
				log.Tracef("filter [%v] not continued", v.Name())
			}
			ctx.AddFlowProcess("skipped")
			break
		}
		if global.Env().IsDebug {
			log.Tracef("processing filter [%v] [%v]", v.Name(), v)
		}
		ctx.AddFlowProcess(v.Name())
		v.Filter(ctx)
	}
}

var nilIDFlowError = errors.New("flow id can't be nil")

func GetFlow(flowID string) (FilterFlow, error) {
	v := FilterFlow{}
	if flowID == "" {
		return v, nilIDFlowError
	}

	v1, ok := flows.Load(flowID)
	if ok {
		return v1.(FilterFlow), nil
	}

	cfg, err := GetFlowConfig(flowID)
	if err != nil {
		return v, err
	}

	if global.Env().IsDebug {
		log.Tracef("flow [%v] [%v]", flowID, cfg)
	}

	if len(cfg.Filters) > 0 {
		flow1, err := pipeline.NewFilter(cfg.GetConfig())
		if flow1 == nil || err != nil {
			return v, err
		}
		v.JoinFilter(flow1)
	}

	flows.Store(flowID, v)
	return v, nil
}

func MustGetFlow(flowID string) FilterFlow {

	flow, err := GetFlow(flowID)
	if err != nil {
		panic(err)
	}
	return flow
}

func GetFlowProcess(flowID string) func(ctx *fasthttp.RequestCtx) {
	flow := MustGetFlow(flowID)
	return flow.Process
}

var flows = &sync.Map{}

var routingRules map[string]RuleConfig = make(map[string]RuleConfig)
var flowConfigs map[string]FlowConfig = make(map[string]FlowConfig)
var routerConfigs map[string]RouterConfig = make(map[string]RouterConfig)

func ClearFlowCache(flow string) {
	flows.Delete(flow)
}

func GetAllFlows() map[string]FilterFlow {
	data := map[string]FilterFlow{}
	flows.Range(func(key, value any) bool {
		data[key.(string)] = value.(FilterFlow)
		return true
	})
	return data
}

func RegisterFlowConfig(flow FlowConfig) {

	if flow.ID == "" && flow.Name != "" {
		flow.ID = flow.Name
	}

	flowConfigs[flow.ID] = flow
	flowConfigs[flow.Name] = flow
	ClearFlowCache(flow.ID)
	ClearFlowCache(flow.Name)
}

func RegisterRouterConfig(config RouterConfig) {
	if config.ID == "" && config.Name != "" {
		config.ID = config.Name
	}
	routerConfigs[config.ID] = config
}

func GetRouter(name string) RouterConfig {
	v, ok := routerConfigs[name]
	if !ok {
		panic(errors.Errorf("router [%s] not found", name))
	}
	return v
}

func GetFlowConfig(id string) (FlowConfig, error) {
	v, ok := flowConfigs[id]
	if !ok {
		return v, errors.Errorf("flow [%s] not found", id)
	}
	return v, nil
}
