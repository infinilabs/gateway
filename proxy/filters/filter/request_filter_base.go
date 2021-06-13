package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"regexp"
)

type RequestFilterBase struct {
	param.Parameters
}

func (filter RequestFilterBase) CheckMustNotRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	var hasRules = false
	arr, ok := filter.GetStringArray("must_not.prefix")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("must_not.contain")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			if util.ContainsAnyInArray(path, arr) {
				if global.Env().IsDebug {
					log.Debugf("hit contain rule [%v] vs [%v]", path, arr)
				}
				return false, hasRules
			}
		}
	}

	arr, ok = filter.GetStringArray("must_not.suffix")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("must_not.wildcard")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			patterns := radix.Compile(arr...)
			ok := patterns.Match(path)
			if ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return false, hasRules
			}
		}
	}

	arr, ok = filter.GetStringArray("must_not.regex")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
					return false, hasRules
				}
			}
		}
	}

	return true, hasRules
}

func (filter RequestFilterBase) CheckMustRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	var hasRules = false
	arr, ok := filter.GetStringArray("must.prefix")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("must.contain")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("must.suffix")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("must.wildcard")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			patterns := radix.Compile(arr...) //TODO handle mutli wildcard rules
			ok := patterns.Match(path)
			if !ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return false, hasRules
			}
		}
	}

	arr, ok = filter.GetStringArray("must.regex")
	if ok {
		if len(arr) > 0 {
			hasRules = true
			for _, v := range arr {
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
	}

	return true, hasRules
}

func (filter RequestFilterBase) CheckShouldRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	var hasShouldRules bool
	arr, ok := filter.GetStringArray("should.prefix")
	if ok {
		if len(arr) > 0 {
			hasShouldRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("should.contain")
	if ok {
		if len(arr) > 0 {
			hasShouldRules = true
			if util.ContainsAnyInArray(path, arr) {
				if global.Env().IsDebug {
					log.Debugf("hit contain rule [%v] vs [%v]", path, arr)
				}
				return true, hasShouldRules
			}
		}
	}

	arr, ok = filter.GetStringArray("should.suffix")
	if ok {
		if len(arr) > 0 {
			hasShouldRules = true
			for _, v := range arr {
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
	}

	arr, ok = filter.GetStringArray("should.wildcard")
	if ok {
		if len(arr) > 0 {
			hasShouldRules = true
			patterns := radix.Compile(arr...)
			ok := patterns.Match(path)
			if ok {
				if global.Env().IsDebug {
					log.Debug("wildcard matched: ", path)
				}
				return true, hasShouldRules
			}
		}
	}

	arr, ok = filter.GetStringArray("should.regex")
	if ok {
		if len(arr) > 0 {
			hasShouldRules = true
			for _, v := range arr {
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
	}
	return false, hasShouldRules
}

func (filter RequestFilterBase) CheckExcludeStringRules(val string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool){
	exclude, ok := filter.GetStringArray("exclude")
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
	if ok {
		hasRule =true
		for _, x := range exclude {
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

func (filter RequestFilterBase) CheckIncludeStringRules(val string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool){
	include, ok := filter.GetStringArray("include")
	if global.Env().IsDebug {
		log.Debug("include:", include)
	}

	if ok {
		hasRule=true
		for _, x := range include {
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

func (filter RequestFilterBase) Filter(ctx *fasthttp.RequestCtx){

	ctx.Response.Header.Set("FILTERED","true")

	if filter.GetStringOrDefault("action","deny") == "deny"{
		ctx.SetDestination("filtered")
		msg,ok:=filter.GetString("message")
		if ok{
			ctx.Response.SwapBody([]byte(msg))
		}

		ctx.Response.Header.Add("original_status",util.IntToString(ctx.Response.StatusCode()))
		ctx.Response.SetStatusCode(filter.GetIntOrDefault("status",403))
		ctx.Finished()
		return
	}

	filterFlow,ok:= filter.GetString("flow")
	if ok{
		ctx.Resume()
		flow := common.MustGetFlow(filterFlow)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.URI().String(), filterFlow, flow.ToString())
		}
		flow.Process(ctx)
		ctx.Finished()
	}
}


//TODO
type RequestUrlQueryArgsFilter struct {
	RequestFilterBase
}

//TODO
type RequestBodyFilter struct {
	RequestFilterBase
}

//TODO
type ResponseBodyFilter struct {
	RequestFilterBase
}

type ResponseContentTypeFilter struct {
	RequestFilterBase
}
