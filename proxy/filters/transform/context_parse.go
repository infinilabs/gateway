/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type ContextParseFilter struct {
	Context string `config:"context"`
	Pattern string `config:"pattern"`
	Group   string `config:"group"`
	p       *regexp.Regexp
}

func (filter *ContextParseFilter) Name() string {
	return "context_parse"
}

func (filter *ContextParseFilter) Filter(ctx *fasthttp.RequestCtx) {
	if filter.Context != "" {
		key, err := ctx.GetValue(filter.Context)
		if err != nil {
			panic(err)
		}
		keyStr := util.ToString(key)
		variables := util.MapStr{}
		if filter.p != nil {
			match := filter.p.FindStringSubmatch(keyStr)
			if len(match)>0{
				for i, name := range filter.p.SubexpNames() {
					if name != "" {
						variables[name] = match[i]
					}
				}
			}
		}
		if len(variables)>0{
			if filter.Group!=""{
				ctx.PutValue(filter.Group,variables)
			}else{
				for k,v:=range variables{
					ctx.PutValue(k,v)
				}
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("context_parse", NewContextParseFilter, &ContextParseFilter{})
}

func NewContextParseFilter(c *config.Config) (pipeline.Filter, error) {
	runner := ContextParseFilter{
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	var err error
	if runner.Context != "" && runner.Pattern != "" {
		runner.p, err = regexp.Compile(runner.Pattern)
		if err != nil {
			panic(err)
		}
	}

	return &runner, nil
}
