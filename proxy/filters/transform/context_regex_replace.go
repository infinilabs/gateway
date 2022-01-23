package transform

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type ContextRegexReplace struct {
	Context string `config:"context"`
	Pattern string `config:"pattern"`
	To      string `config:"to"`
	p       *regexp.Regexp
}

func (filter *ContextRegexReplace) Name() string {
	return "context_regex_replace"
}

func (filter *ContextRegexReplace) Filter(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Trace("context:", filter.Context, "pattern:", filter.Pattern, ", to:", filter.To)
	}

	if filter.Context != "" {
		value, err := ctx.GetValue(filter.Context)
		if err != nil {
			log.Error(err)
			return
		}
		valueStr := util.ToString(value)
		if len(valueStr) > 0 {
			newBody := filter.p.ReplaceAll([]byte(valueStr), util.UnsafeStringToBytes(filter.To))
			err := ctx.SetValue(filter.Context, string(newBody))
			if err != nil {
				log.Error(err)
				return
			}
		}
	}
}

func NewContextRegexReplace(c *config.Config) (filter pipeline.Filter, err error) {

	runner := ContextRegexReplace{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.p, err = regexp.Compile(runner.Pattern)
	if err != nil {
		panic(err)
	}
	return &runner, nil
}
