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
	Message string    `config:"message"`
	Rules []MatchRules    `config:"rules"`
}

func NewRequestPathLimitFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RequestPathLimitFilter {
		Message: "Reach request limit!",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	for _,v:=range runner.Rules{
		if !v.Valid(){
			panic(errors.Errorf("invalid pattern:%v",v))
		}
	}

	return &runner, nil
}


func (filter *RequestPathLimitFilter) Name() string {
	return "request_path_limiter"
}

type MatchRules struct {
	Pattern string  `config:"pattern"` //pattern
	MaxQPS  int64   `config:"max_qps"`//max_qps
	reg     *regexp.Regexp
	ExtractGroup string `config:"group"`
}

func (this *MatchRules)Extract(input string)string  {
	match := this.reg.FindStringSubmatch(input)
	for i, name := range this.reg.SubexpNames() {
		if name==this.ExtractGroup{
			return match[i]
		}
	}
	return ""
}

func (this *MatchRules)Match(input string)bool  {
	return this.reg.MatchString(input)
}

func (this *MatchRules) Valid()bool {

	if this.MaxQPS<=0{
		log.Warnf("invalid throttle rule, pattern:[%v] group:[%v] max_qps:[%v], reset max_qps to 10,000",this.Pattern,this.ExtractGroup,this.MaxQPS)
		this.MaxQPS=10000
	}

	reg,err:= regexp.Compile(this.Pattern)
	if err!=nil{
		return false
	}

	if this.ExtractGroup==""{
		return false
	}

	if this.reg==nil{
		this.reg=reg
	}

	return true
}

func (filter *RequestPathLimitFilter) Filter(ctx *fasthttp.RequestCtx) {

	key:=string(ctx.Path())

	for _,v:=range filter.Rules{
		if v.Match(key){
			item:=v.Extract(key)

			if global.Env().IsDebug{
				log.Debug(key," matches ",v.Pattern,", extract:",item)
			}

			if item!=""{
				if !rate.GetRateLimiterPerSecond(v.Pattern,item, int(v.MaxQPS)).Allow(){

					if global.Env().IsDebug{
						log.Debug(key," reach limited ",v.Pattern,",extract:",item)
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
