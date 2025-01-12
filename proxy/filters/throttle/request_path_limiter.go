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

package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
)

type RequestPathLimitFilter struct {
	WarnMessage bool          `config:"log_warn_message"`
	Message     string        `config:"message"`
	Rules       []*MatchRules `config:"rules"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_path_limiter", NewRequestPathLimitFilter, &RequestPathLimitFilter{})
}

func NewRequestPathLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestPathLimitFilter{
		Message: "Reach request limit!",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	for _, v := range runner.Rules {
		if !v.Valid() {
			panic(errors.Errorf("invalid pattern:%v", v))
		}
	}

	return &runner, nil
}

func (filter *RequestPathLimitFilter) Name() string {
	return "request_path_limiter"
}

type MatchRules struct {
	Pattern      string `config:"pattern"` //pattern
	MaxQPS       int64  `config:"max_qps"` //max_qps
	reg          *regexp.Regexp
	ExtractGroup string `config:"group"`
}

func (this *MatchRules) Extract(input string) string {
	match := this.reg.FindStringSubmatch(input)
	for i, name := range this.reg.SubexpNames() {
		if name == this.ExtractGroup {
			return match[i]
		}
	}
	return ""
}

func (this *MatchRules) Match(input string) bool {
	return this.reg.MatchString(input)
}

func (this *MatchRules) Valid() bool {

	if this.MaxQPS <= 0 {
		log.Warnf("invalid throttle rule, pattern:[%v] group:[%v] max_qps:[%v], reset max_qps to 10,000", this.Pattern, this.ExtractGroup, this.MaxQPS)
		this.MaxQPS = 10000
	}

	reg, err := regexp.Compile(this.Pattern)
	if err != nil {
		return false
	}

	if this.ExtractGroup == "" {
		return false
	}

	if this.reg == nil {
		this.reg = reg
	}

	return true
}

func (filter *RequestPathLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	key := string(ctx.Path())

	for _, v := range filter.Rules {
		if v.Match(key) {
			item := v.Extract(key)

			if global.Env().IsDebug {
				log.Debug(key, " matches ", v.Pattern, ", extract:", item)
			}

			if item != "" {
				if !rate.GetRateLimiterPerSecond(v.Pattern, item, int(v.MaxQPS)).Allow() {

					if global.Env().IsDebug {
						log.Debug(key, " reach limited ", v.Pattern, ",extract:", item)
					}

					if filter.WarnMessage {
						log.Warnf("request throttled: %v", string(ctx.Path()))
					}

					ctx.SetStatusCode(429)
					ctx.WriteString(filter.Message)
					ctx.Finished()
				}
				break
			}
		}
	}

}
