package filter

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"regexp"
)

type RequestFilter struct {
	Action     string `config:"action"`
	Message     string `config:"message"`
	Status     int `config:"status"`
	Flow     string `config:"flow"`
	Rules config.Rules `config:"rules"`
	Include []string `config:"include"`
	Exclude []string `config:"exclude"`
}
//
//func NewRequestFilter(c *config.Config) (pipeline.Filter, error) {
//
//	runner := RequestFilter {
//		Action: "deny",
//		Status:403,
//	}
//
//	if err := c.Unpack(&runner); err != nil {
//		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
//	}
//
//	return &runner, nil
//}

func (filter RequestFilter) Name() string {
	return "request_filter"
}

func (filter *RequestFilter) CheckMustNotRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	var hasRules = false

	if filter.Rules.MustNot!=nil{
		return true,false
	}

	if len(filter.Rules.MustNot.Prefix)>0 {
		hasRules = true
		for _, v := range filter.Rules.MustNot.Prefix {
			if global.Env().IsDebug {
				log.Tracef("check prefix rule [%v] vs [%v]", path, v)
			}
			if util.PrefixStr(path, v) {
				if global.Env().IsDebug {
					log.Debugf("hit prefix rule [%v] vs [%v]", path, v)
				}
				return false, hasRules
			}
		}
	}

	if len(filter.Rules.MustNot.Contain)>0 {
		hasRules = true
		if util.ContainsAnyInArray(path, filter.Rules.MustNot.Contain) {
			if global.Env().IsDebug {
				log.Debugf("hit contain rule [%v] vs [%v]", path, filter.Rules.MustNot.Contain)
			}
			return false, hasRules
		}
	}

	if len(filter.Rules.MustNot.Suffix)>0 {
		hasRules = true
			for _, v := range filter.Rules.MustNot.Suffix {
				if global.Env().IsDebug {
					log.Tracef("check suffix rule [%v] vs [%v]", path, v)
				}
				if util.SuffixStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("hit suffix rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
	}

	if len(filter.Rules.MustNot.Wildcard)>0{
		hasRules = true
			patterns := radix.Compile(filter.Rules.MustNot.Wildcard...)
			ok := patterns.Match(path)
			if ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return false, hasRules
			}
	}


	if len(filter.Rules.MustNot.Regex)>0{
			hasRules = true
			for _, v := range filter.Rules.MustNot.Regex {
				if global.Env().IsDebug {
					log.Tracef("check regex rule [%v] vs [%v]", path, v)
				}
				//TODO reuse regexp
				reg, err := regexp.Compile(v)
				if err != nil {
					panic(err)
				}
				if reg.MatchString(path) {
					if global.Env().IsDebug {
						log.Debugf("hit regex rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
	}

	return true, hasRules
}

func (filter *RequestFilter) CheckMustRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {

	if filter.Rules.Must!=nil{
		return true,false
	}

	var hasRules = false
	if len(filter.Rules.Must.Prefix)>0 {
			hasRules = true
			for _, v := range filter.Rules.Must.Prefix {
				if global.Env().IsDebug {
					log.Tracef("check prefix rule [%v] vs [%v]", path, v)
				}
				if !util.PrefixStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("not match prefix rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
	}

	if len(filter.Rules.Must.Contain)>0 {
			hasRules = true
			for _, v := range filter.Rules.Must.Contain {
				if global.Env().IsDebug {
					log.Tracef("check contain rule [%v] vs [%v]", path, v)
				}
				if !util.ContainStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("not match contain rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
		}

	if len(filter.Rules.Must.Suffix)>0 {
			hasRules = true
			for _, v := range filter.Rules.Must.Suffix {
				if global.Env().IsDebug {
					log.Tracef("check suffix rule [%v] vs [%v]", path, v)
				}
				if !util.SuffixStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("not match suffix rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
		}


	if len(filter.Rules.Must.Wildcard)>0 {

			hasRules = true
			patterns := radix.Compile(filter.Rules.Must.Wildcard...) //TODO handle mutli wildcard rules
			ok := patterns.Match(path)
			if !ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return false, hasRules
			}
		}

	if len(filter.Rules.Must.Regex)>0 {
			hasRules = true
			for _, v := range filter.Rules.Must.Regex {
				if global.Env().IsDebug {
					log.Tracef("check regex rule [%v] vs [%v]", path, v)
				}
				reg, err := regexp.Compile(v)
				if err != nil {
					panic(err)
				}
				if !reg.MatchString(path) {
					if global.Env().IsDebug {
						log.Debugf("not match regex rule [%v] vs [%v]", path, v)
					}
					return false, hasRules
				}
			}
		}

	return true, hasRules
}

func (filter *RequestFilter) CheckShouldRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {

	if filter.Rules.Should!=nil{
		return true,false
	}

	var hasShouldRules bool
	if len(filter.Rules.Should.Prefix)>0 {
			hasShouldRules = true
			for _, v := range filter.Rules.Should.Prefix {
				if global.Env().IsDebug {
					log.Tracef("check prefix rule [%v] vs [%v]", path, v)
				}
				if util.PrefixStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("hit prefix rule [%v] vs [%v]", path, v)
					}
					return true, hasShouldRules
				}
			}
		}


	if len(filter.Rules.Should.Contain)>0 {
			hasShouldRules = true
			if util.ContainsAnyInArray(path, filter.Rules.Should.Contain) {
				if global.Env().IsDebug {
					log.Debugf("hit contain rule [%v] vs [%v]", path, filter.Rules.Should.Contain)
				}
				return true, hasShouldRules
			}
	}

	if len(filter.Rules.Should.Suffix)>0 {
			hasShouldRules = true
			for _, v := range filter.Rules.Should.Suffix {
				if global.Env().IsDebug {
					log.Tracef("check suffix rule [%v] vs [%v]", path, v)
				}
				if util.SuffixStr(path, v) {
					if global.Env().IsDebug {
						log.Debugf("hit suffix rule [%v] vs [%v]", path, v)
					}
					return true, hasShouldRules
				}
			}
	}

	if len(filter.Rules.Should.Wildcard)>0 {
			hasShouldRules = true
			patterns := radix.Compile(filter.Rules.Should.Wildcard...)
			ok := patterns.Match(path)
			if ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return true, hasShouldRules
			}
	}

	if len(filter.Rules.Should.Regex)>0 {
			hasShouldRules = true
			for _, v := range filter.Rules.Should.Regex {
				if global.Env().IsDebug {
					log.Tracef("check regex rule [%v] vs [%v]", path, v)
				}
				reg, err := regexp.Compile(v)
				if err != nil {
					panic(err)
				}
				if reg.MatchString(path) {
					if global.Env().IsDebug {
						log.Debugf("hit regex rule [%v] vs [%v]", path, v)
					}
					return true, hasShouldRules
				}
			}
		}
	return false, hasShouldRules
}

func (filter *RequestFilter) CheckExcludeStringRules(val string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool){
	if global.Env().IsDebug {
		log.Debug("exclude:", filter.Exclude)
	}
	if len(filter.Exclude)>0 {
		hasRule =true
		for _, x := range filter.Exclude {
			match := x == val
			if global.Env().IsDebug {
				log.Debugf("check exclude rule: %v vs %v, match: %v", x, val, match)
			}
			if match {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return false,hasRule
			}
		}
	}

	return true,hasRule
}

func (filter *RequestFilter) CheckIncludeStringRules(val string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool){
	if global.Env().IsDebug {
		log.Debug("include:", filter.Include)
	}

	if len(filter.Include)>0 {
		hasRule=true
		for _, x := range filter.Include {
			match := x == val
			if global.Env().IsDebug {
				log.Debugf("check include rule: %v vs %v, match: %v", x, val, match)
			}
			if match {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
				}
				return true,hasRule
			}
		}
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return false,hasRule
	}

	return !hasRule, hasRule

}

func (filter *RequestFilter) Filter(ctx *fasthttp.RequestCtx){

	ctx.Response.Header.Set("FILTERED","true")

	if filter.Action == "deny"{
		ctx.SetDestination("filtered")
		if len(filter.Message)>0{
			ctx.SetContentType(util.ContentTypeJson)
			ctx.Response.SwapBody([]byte(fmt.Sprintf("{\"error\":true,\"message\":\"%v\"}",filter.Message)))
		}

		ctx.Response.Header.Add("original_status",util.IntToString(ctx.Response.StatusCode()))
		ctx.Response.SetStatusCode(filter.Status)
		ctx.Finished()
		return
	}

	if filter.Flow!=""{
		ctx.Resume()
		flow := common.MustGetFlow(filter.Flow)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), filter.Flow, flow.ToString())
		}
		flow.Process(ctx)
		ctx.Finished()
	}
}
