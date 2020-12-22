package filters

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type RequestFilterBase struct {
	param.Parameters
}

type RequestHeaderFilter struct {
	RequestFilterBase
}

func (filter RequestHeaderFilter) Name() string {
	return "request_header_filter"
}

func (filter RequestHeaderFilter) Process(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Debug("headers:", string(util.EscapeNewLine(ctx.Request.Header.Header())))
	}

	exclude, ok := filter.GetMapArray("exclude")
	if ok {
		for _, x := range exclude {
			for k, v := range x {
				v1 := ctx.Request.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("exclude header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					ctx.Filtered()
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
	}

	include, ok := filter.GetMapArray("include")
	if ok {
		for _, x := range include {
			for k, v := range x {
				v1 := ctx.Request.Header.Peek(k)
				match := util.ToString(v) == string(v1)
				if global.Env().IsDebug {
					log.Debugf("include header [%v]: %v vs %v, match: %v", k, v, string(v1), match)
				}
				if match {
					if global.Env().IsDebug {
						log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
					}
					return
				}
			}
		}
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}

}

type RequestMethodFilter struct {
	RequestFilterBase
}

func (filter RequestMethodFilter) Name() string {
	return "request_method_filter"
}

func (filter RequestMethodFilter) Process(ctx *fasthttp.RequestCtx) {

	method := string(ctx.Method())

	if global.Env().IsDebug {
		log.Debug("method:", method)
	}

	exclude, ok := filter.GetStringArray("exclude")
	if ok {
		for _, x := range exclude {
			if global.Env().IsDebug {
				log.Debugf("exclude method: %v vs %v, match: %v", x, method, util.ToString(x) == method)
			}
			if util.ToString(x) == method {
				ctx.Filtered()
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	include, ok := filter.GetStringArray("include")
	if ok {
		for _, x := range include {
			if global.Env().IsDebug {
				log.Debugf("include method [%v]: %v vs %v, match: %v", x, method, util.ToString(x) == string(method))
			}
			if util.ToString(x) == method {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.URI().String())
				}
				return
			}
		}
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}

}

type RequestUrlPathFilter struct {
	RequestFilterBase
}

func (filter RequestUrlPathFilter) Name() string {
	return "request_path_filter"
}

func (filter RequestUrlPathFilter) Process(ctx *fasthttp.RequestCtx) {

	path := string(ctx.Path())

	//TODO check cache first

	if global.Env().IsDebug {
		log.Debug("path:", path)
	}

	var hasOtherRules = false
	var hasRules = false
	var valid = false
	valid, hasRules = filter.CheckMustNotRules(path, ctx)
	if !valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	valid, hasRules = filter.CheckMustRules(path, ctx)

	if !valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must rules not matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	if hasRules {
		hasOtherRules = true
	}

	var hasShouldRules = false
	valid, hasShouldRules = filter.CheckShouldRules(path, ctx)
	if !valid {
		if !hasOtherRules && hasShouldRules {
			ctx.Filtered()
			if global.Env().IsDebug {
				log.Debugf("only should rules, but none of them are matched, this request has been filtered: %v", ctx.Request.URI().String())
			}
		}
	}

	//	rules=&config.Rules{
	//		Must: &config.Rule{},
	//		MustNot: &config.Rule{},
	//		Should: &config.Rule{},
	//	}
	//filter.Config("must",rules.Must)

	//fmt.Println(util.ToJson(filter.Data,true))
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


//TODO
type RequestUrlQueryArgsFilter struct {
	RequestFilterBase
}

//TODO
type RequestBodyFilter struct {
	RequestFilterBase
}

//TODO
type ResponseCodeFilter struct {
	RequestFilterBase
}

//TODO
type ResponseHeaderFilter struct {
	RequestFilterBase
}

//TODO
type ResponseBodyFilter struct {
	RequestFilterBase
}
