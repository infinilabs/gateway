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

package filter

import (
	"fmt"
	"regexp"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/radix"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
)

type RequestFilter struct {
	Action  string       `config:"action"`
	Message string       `config:"message"`
	Status  int          `config:"status"`
	Flow    string       `config:"flow"`
	Should  *config.Rule `config:"should"`
	Must    *config.Rule `config:"must"`
	MustNot *config.Rule `config:"must_not"`
}

func (filter RequestFilter) Name() string {
	return "request_filter"
}

func (filter *RequestFilter) CheckMustNotRules(path string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	var hasRules = false

	if filter.MustNot == nil {
		return true, false
	}

	if len(filter.MustNot.Prefix) > 0 {
		hasRules = true
		for _, v := range filter.MustNot.Prefix {
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

	if len(filter.MustNot.Contain) > 0 {
		hasRules = true
		if util.ContainsAnyInArray(path, filter.MustNot.Contain) {
			if global.Env().IsDebug {
				log.Debugf("hit contain rule [%v] vs [%v]", path, filter.MustNot.Contain)
			}
			return false, hasRules
		}
	}

	if len(filter.MustNot.Suffix) > 0 {
		hasRules = true
		for _, v := range filter.MustNot.Suffix {
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

	if len(filter.MustNot.Wildcard) > 0 {
		hasRules = true
		patterns := radix.Compile(filter.MustNot.Wildcard...)
		ok := patterns.Match(path)
		if ok {
			if global.Env().IsDebug {
				log.Debug("wildcard matched: ", path)
			}
			return false, hasRules
		}
	}

	if len(filter.MustNot.Regex) > 0 {
		hasRules = true
		for _, v := range filter.MustNot.Regex {
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

	if filter.Must == nil {
		return true, false
	}

	var hasRules = false
	if len(filter.Must.Prefix) > 0 {
		hasRules = true
		for _, v := range filter.Must.Prefix {
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

	if len(filter.Must.Contain) > 0 {
		hasRules = true
		for _, v := range filter.Must.Contain {
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

	if len(filter.Must.Suffix) > 0 {
		hasRules = true
		for _, v := range filter.Must.Suffix {
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

	if len(filter.Must.Wildcard) > 0 {

		hasRules = true
		patterns := radix.Compile(filter.Must.Wildcard...) //TODO handle mutli wildcard rules
		ok := patterns.Match(path)
		if !ok {
			if global.Env().IsDebug {
				log.Debug("wildcard matched: ", path)
			}
			return false, hasRules
		}
	}

	if len(filter.Must.Regex) > 0 {
		hasRules = true
		for _, v := range filter.Must.Regex {
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

	if filter.Should == nil {
		return true, false
	}

	var hasShouldRules bool
	if len(filter.Should.Prefix) > 0 {
		hasShouldRules = true
		for _, v := range filter.Should.Prefix {
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

	if len(filter.Should.Contain) > 0 {
		hasShouldRules = true
		if util.ContainsAnyInArray(path, filter.Should.Contain) {
			if global.Env().IsDebug {
				log.Debugf("hit contain rule [%v] vs [%v]", path, filter.Should.Contain)
			}
			return true, hasShouldRules
		}
	}

	if len(filter.Should.Suffix) > 0 {
		hasShouldRules = true
		for _, v := range filter.Should.Suffix {
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

	if len(filter.Should.Wildcard) > 0 {
		hasShouldRules = true
		patterns := radix.Compile(filter.Should.Wildcard...)
		ok := patterns.Match(path)
		if ok {
			if global.Env().IsDebug {
				log.Debug("wildcard matched: ", path)
			}
			return true, hasShouldRules
		}
	}

	if len(filter.Should.Regex) > 0 {
		hasShouldRules = true
		for _, v := range filter.Should.Regex {
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

func CheckExcludeStringRules(val string, array []string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	if global.Env().IsDebug {
		log.Debug("exclude:", array)
	}
	if len(array) > 0 {
		hasRule = true
		for _, x := range array {
			match := x == val
			if global.Env().IsDebug {
				log.Debugf("check exclude rule: %v vs %v, match: %v", x, val, match)
			}
			if match {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
				}
				return false, hasRule
			}
		}
	}

	return true, hasRule
}

func CheckIncludeStringRules(val string, array []string, ctx *fasthttp.RequestCtx) (valid bool, hasRule bool) {
	if global.Env().IsDebug {
		log.Debug("include:", array)
	}

	if len(array) > 0 {
		hasRule = true
		for _, x := range array {
			match := x == val
			if global.Env().IsDebug {
				log.Debugf("check include rule: %v vs %v, match: %v", x, val, match)
			}
			if match {
				if global.Env().IsDebug {
					log.Debugf("rule matched, this request has been marked as good one: %v", ctx.Request.PhantomURI().String())
				}
				return true, hasRule
			}
		}
		if global.Env().IsDebug {
			log.Debugf("no rule matched, this request has been filtered: %v", ctx.Request.PhantomURI().String())
		}
		return false, hasRule
	}

	return !hasRule, hasRule

}

func (filter *RequestFilter) Filter(ctx *fasthttp.RequestCtx) {

	ctx.Response.Header.Set("FILTERED", "true")

	if filter.Action == "deny" {
		ctx.SetDestination("filtered")
		if len(filter.Message) > 0 {
			ctx.SetContentType(util.ContentTypeJson)
			ctx.Response.SwapBody([]byte(fmt.Sprintf("{\"error\":true,\"message\":\"%v\"}", filter.Message)))
		}

		ctx.Response.Header.Set("original_status", util.IntToString(ctx.Response.StatusCode()))
		ctx.Response.SetStatusCode(filter.Status)
		ctx.Finished()
		return
	}

	if filter.Flow != "" {
		ctx.Resume()
		flow := common.MustGetFlow(filter.Flow)
		if global.Env().IsDebug {
			log.Debugf("request [%v] go on flow: [%s] [%s]", ctx.PhantomURI().String(), filter.Flow, flow.ToString())
		}
		flow.Process(ctx)
		ctx.Finished()
	}
}
