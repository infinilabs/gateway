package throttle

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/lib/fasthttp"
	"regexp"
	log "github.com/cihub/seelog"
)

type RequestPathLimitFilter struct {
	param.Parameters
}

func (filter RequestPathLimitFilter) Name() string {
	return "request_path_limiter"
}

var inited bool

type MatchRules struct {
	Pattern string //pattern
	MaxQPS  int64 //max_qps
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

func (filter RequestPathLimitFilter) Process(ctx *fasthttp.RequestCtx) {

	if !inited{
		results:=[]MatchRules{}
		rules:=filter.Get("rules")
		objs:=rules.([]interface{})
		for _,v:=range objs{
			x:=v.(map[string]interface{})
			z:=MatchRules{}
			z.Pattern=x["pattern"].(string)
			z.ExtractGroup = x["group"].(string)
			z.MaxQPS,_= param.GetInt64OrDefault(x["max_qps"],-1024)
			if !z.Valid(){
				log.Warnf("invalid throttle rule, pattern:[%v] group:[%v] max_qps:[%v], skipping",z.Pattern,z.ExtractGroup,z.MaxQPS)
				continue
			}
			results=append(results,z)
		}
		filter.Set("rules_obj",results)
		inited=true
	}

	rules,ok:=filter.Get("rules_obj").([]MatchRules)
	if !ok{
		return
	}


	key:=string(ctx.Path())

	if global.Env().IsDebug{
		log.Debug(len(rules)," rules,",key)
	}

	for _,v:=range rules{
		if v.Match(key){
			item:=v.Extract(key)

			if global.Env().IsDebug{
				log.Debug(key," matches ",v.Pattern,", extract:",item)
			}

			if item!=""{
				if !rate.GetRaterWithDefine(v.Pattern,item, int(v.MaxQPS)).Allow(){

					if global.Env().IsDebug{
						log.Debug(key," reach limited ",v.Pattern,",extract:",item)
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
