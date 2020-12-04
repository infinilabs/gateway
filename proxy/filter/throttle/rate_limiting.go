package throttle

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
	log "src/github.com/cihub/seelog"
)

type RateLimitFilter struct {
	param.Parameters
}

func (filter RateLimitFilter) Name() string {
	return "rate_limit"
}

var inited bool

type MatchRules struct {
	Pattern string //pattern
	MaxQPS  int //max_qps
	reg     *regexp.Regexp
	ExtractGroup string
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
	if this.reg==nil{
		this.reg= regexp.MustCompile(this.Pattern)
	}
	return this.reg.MatchString(input)
}

func (filter RateLimitFilter) Process(ctx *fasthttp.RequestCtx) {

	if !inited{
		results:=[]MatchRules{}
		rules:=filter.Get("rules")
		objs:=rules.([]interface{})
		for _,v:=range objs{
			x:=v.(map[string]interface{})
			z:=MatchRules{}
			z.Pattern=x["pattern"].(string)
			z.MaxQPS = int(x["max_qps"].(uint64))
			z.ExtractGroup = x["group"].(string)
			results=append(results,z)
		}
		filter.Set("rules_obj",results)
		inited=true
	}

	rules:=filter.Get("rules_obj").([]MatchRules)



	path:=ctx.URI().Path()
	key:=string(path)

	if global.Env().IsDebug{
		log.Debug(len(rules)," rules,",key)
	}

	for _,v:=range rules{
		if v.Match(key){
			item:=v.Extract(key)

			if global.Env().IsDebug{
				log.Debug(key," matches ",v.Pattern,"extract:",item)
			}

			if item!=""{
				if !rate.GetRaterWithDefine(v.Pattern,item, int(v.MaxQPS)).Allow(){

					if global.Env().IsDebug{
						log.Debug(key," reach limited ",v.Pattern,"extract:",item)
					}

					ctx.SetStatusCode(429)
					ctx.WriteString(filter.GetStringOrDefault("message","Reach request limit!"))
					ctx.Finished()
				}
				break
			}
		}
	}

}
