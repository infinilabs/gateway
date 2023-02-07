package routing

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"io"
	"github.com/valyala/fasttemplate"
	"regexp"
	"time"
)

type FlowFilter struct {
	Flow                string   `config:"flow"`
	Flows               []string `config:"flows"`
	IgnoreUndefinedFlow bool     `config:"ignore_undefined_flow"`
	items               []FlowItem

	ContextFlow *ContextFlow `config:"context_flow"`
	cache       *util.Cache
}

type ContextFlow struct {
	Context  string `config:"context"`
	Pattern  string `config:"context_parse_pattern"`
	Template string `config:"flow_id_template"`
	Continue bool     `config:"continue"`

	p *regexp.Regexp
	template    *fasttemplate.Template
	hasTemplate bool
}

type FlowItem struct {
	Flow        string `config:"flow"`
	template    *fasttemplate.Template
	hasTemplate bool
}

func (filter *FlowFilter) Name() string {
	return "flow"
}

func (filter *FlowFilter) Filter(ctx *fasthttp.RequestCtx) {

	if filter.ContextFlow != nil {
		key, err := ctx.GetValue(filter.ContextFlow.Context)
		if err != nil {
			panic(err)
		}

		flowID:=filter.ContextFlow.Template
		var hitCache=false
		//check cache first
		if filter.cache!=nil{
			v:=filter.cache.Get(key)
			if v!=nil{
				tmp,ok:=v.(string)
				if ok{
					flowID=tmp
					hitCache=true
				}
			}
		}

		if !hitCache{
			keyStr := util.ToString(key)
			variables := util.MapStr{}
			if filter.ContextFlow.p != nil {
				match := filter.ContextFlow.p.FindStringSubmatch(keyStr)
				if len(match)>0{
					for i, name := range filter.ContextFlow.p.SubexpNames() {
						if name != "" {
							variables[name] = match[i]
						}
					}
				}
			}



			if filter.ContextFlow.hasTemplate {

				if global.Env().IsDebug {
					log.Tracef("variable: %v", variables)
				}

				flowID = filter.ContextFlow.template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {

					variable,ok:=variables[tag]
					if ok{
						return w.Write([]byte(util.ToString(variable)))
					}

					variable, err := ctx.GetValue(tag)
					if err != nil {
						panic(errors.Wrap(err,"tag was not found in context"))
					}
					return w.Write([]byte(util.ToString(variable)))
				})

				if global.Env().IsDebug {
					log.Debugf("flow_id: %v -> %v", filter.ContextFlow.Template,flowID)
				}
			}
		}

		flow, err := common.GetFlow(util.ToString(flowID))
		if err != nil {
			log.Errorf("failed to get flow [%v], err: [%v], continue", flowID, err)
			if !filter.IgnoreUndefinedFlow {
				panic(err)
			}
		} else {
			if global.Env().IsDebug {
				log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), flowID, flow.ToString())
			}
			ctx.AddFlowProcess("flow:" + flow.ID)
			flow.Process(ctx)

			//update cache
			if filter.cache!=nil&&!hitCache{
				filter.cache.Put(key,flowID)
			}

			if !filter.ContextFlow.Continue{
				ctx.Finished()
				return
			}
		}
	}

	for _, v := range filter.items {

		flowID := v.Flow
		if v.hasTemplate {
			flowID = v.template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
				variable, err := ctx.GetValue(tag)
				if err != nil {
					panic(err)
				}
				return w.Write([]byte(util.ToString(variable)))
			})

			if global.Env().IsDebug {
				log.Tracef("flow [%v] contains template, rendering to: %v", v, flowID)
			}
		}

		flow, err := common.GetFlow(flowID)
		if err != nil {
			log.Errorf("failed to get flow [%v], err: [%v], continue", flowID, err)
			if  !filter.IgnoreUndefinedFlow{
				panic(err)
			}
		}else{
			if global.Env().IsDebug {
				log.Tracef("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), v, flow.ToString())
			}

			ctx.AddFlowProcess("flow:" + flow.ID)
			flow.Process(ctx)
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("flow", NewFlowFilter, &FlowFilter{})
}

func NewFlowFilter(c *config.Config) (pipeline.Filter, error) {

	runner := FlowFilter{
		IgnoreUndefinedFlow: true,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if runner.Flow != "" {
		runner.Flows = append(runner.Flows, runner.Flow)
	}

	runner.items = []FlowItem{}
	var err error
	for _, v := range runner.Flows {

		item := FlowItem{}
		item.Flow = v
		if util.ContainStr(v, "$[[") {
			if global.Env().IsDebug {
				log.Tracef("flow [%v] contains template, rendering now", v)
			}
			item.template, err = fasttemplate.NewTemplate(v, "$[[", "]]")
			if err != nil {
				panic(err)
			}
			item.hasTemplate = true
		}
		runner.items = append(runner.items, item)
	}

	if runner.ContextFlow != nil {
		//init regexp
		if runner.ContextFlow.Pattern != "" {
			runner.ContextFlow.p, err = regexp.Compile(runner.ContextFlow.Pattern)
			if err != nil {
				panic(err)
			}
		}

		//init template
		if util.ContainStr(runner.ContextFlow.Template, "$[[") {
			if global.Env().IsDebug {
				log.Tracef("flow [%v] contains template, rendering now", runner.ContextFlow.Template)
			}
			runner.ContextFlow.template, err = fasttemplate.NewTemplate(runner.ContextFlow.Template, "$[[", "]]")
			if err != nil {
				panic(err)
			}
			runner.ContextFlow.hasTemplate = true
		}
	}

	runner.cache=util.NewCacheWithExpireOnAdd(600*time.Second,100)


	return &runner, nil
}
