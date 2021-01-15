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
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
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
				log.Debugf("include method: %v vs %v, match: %v", x, method, util.ToString(x) == string(method))
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
	if global.Env().IsDebug {
		log.Debug("include:", exclude)
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


type ResponseStatusCodeFilter struct {
	RequestFilterBase
}

func (filter ResponseStatusCodeFilter) Name() string {
	return "response_status_filter"
}

func (filter ResponseStatusCodeFilter) Process(ctx *fasthttp.RequestCtx) {

	code := ctx.Response.StatusCode()

	if global.Env().IsDebug {
		log.Debug("code:", code)
	}

	exclude, ok := filter.GetInt64Array("exclude")
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
	if ok {
		for _, x := range exclude {
			y:=int(x)
			if global.Env().IsDebug {
				log.Debugf("exclude code: %v vs %v, match: %v", x, code,  y== code)
			}
			if y == code {
				ctx.Filtered()
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.URI().String())
				}
				return
			}
		}
	}

	include, ok := filter.GetInt64Array("include")
	if global.Env().IsDebug {
		log.Debug("include:", exclude)
	}
	if ok {
		for _, x := range include {
			y:=int(x)
			if global.Env().IsDebug {
				log.Debugf("include code: %v vs %v, match: %v", x, code, y == code)
			}
			if y == code {
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

type ResponseHeaderFilter struct {
	RequestFilterBase
}

func (filter ResponseHeaderFilter) Name() string {
	return "response_header_filter"
}

func (filter ResponseHeaderFilter) Process(ctx *fasthttp.RequestCtx) {

	if global.Env().IsDebug {
		log.Debug("headers:", string(util.EscapeNewLine(ctx.Response.Header.Header())))
	}

	exclude, ok := filter.GetMapArray("exclude")
	if ok {
		for _, x := range exclude {
			for k, v := range x {
				v1 := ctx.Response.Header.Peek(k)
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
				v1 := ctx.Response.Header.Peek(k)
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



type RequestClientIPFilter struct {
	RequestFilterBase
}


func (filter RequestClientIPFilter) Name() string {
	return "request_client_ip_filter"
}

func (filter RequestClientIPFilter) Process(ctx *fasthttp.RequestCtx) {

	clientIP:=ctx.RemoteIP().String()
	if global.Env().IsDebug {
		log.Debug("client_ip:", clientIP)
	}

	exclude, ok := filter.GetStringArray("exclude")
	if global.Env().IsDebug {
		log.Debug("exclude:", exclude)
	}
	if ok {
		for _, x := range exclude {
				match := x == clientIP
				if global.Env().IsDebug {
					log.Debugf("exclude clientIP %v vs %v, match: %v", x, clientIP, match)
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

	include, ok := filter.GetStringArray("include")
	if global.Env().IsDebug {
		log.Debug("include:", include)
	}
	if ok {
		for _, x := range include {
				match := x == clientIP
				if global.Env().IsDebug {
					log.Debugf("include clientIP %v vs %v, match: %v", x, clientIP, match)
				}
				if match {
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
				ctx.Filtered()
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
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
	}

	return !hasRule, hasRule

}

type RequestUserFilter struct {
	RequestFilterBase
}

func (filter RequestUserFilter) Name() string {
	return "request_user_filter"
}

func (filter RequestUserFilter) Process(ctx *fasthttp.RequestCtx) {
	exists,user,_:=ctx.ParseBasicAuth()
	if !exists{
		if global.Env().IsDebug{
			log.Tracef("user not exist")
		}
		return
	}

	userStr:=string(user)
	valid, hasRule:= filter.CheckExcludeStringRules(userStr, ctx)
	if hasRule&&!valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(userStr, ctx)
	if hasRule&&!valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

}

type RequestAPIKeyFilter struct {
	RequestFilterBase
}

type RequestServerHostFilter struct {
	RequestFilterBase
}

func (filter RequestServerHostFilter) Name() string {
	return "request_host_filter"
}

func (filter RequestServerHostFilter) Process(ctx *fasthttp.RequestCtx) {
	host:=string(ctx.Request.Host())
	valid, hasRule:= filter.CheckExcludeStringRules(host, ctx)
	if hasRule&&!valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
	}

	valid, hasRule= filter.CheckIncludeStringRules(host, ctx)
	if hasRule&&!valid {
		ctx.Filtered()
		if global.Env().IsDebug {
			log.Debugf("must_not rules matched, this request has been filtered: %v", ctx.Request.URI().String())
		}
		return
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
