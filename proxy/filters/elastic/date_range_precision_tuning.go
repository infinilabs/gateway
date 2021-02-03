package elastic

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type DatePrecisionTuning struct {
	param.Parameters
}

func (this DatePrecisionTuning) Name() string {
	return "date_range_precision_tuning"
}

var builtinKeywords=[]string{"_search","_async_search"}
func (this DatePrecisionTuning) Process(ctx *fasthttp.RequestCtx) {

	if ctx.Request.GetBodyLength()<=0{
		return
	}

	path:=string(ctx.RequestURI())
	keywords,ok:=this.GetStringArray("path_keywords")
	if !ok{
		keywords=builtinKeywords
	}

	if util.ContainsAnyInArray(path,keywords){
			//request normalization
			//timestamp precision processing, scale time from million seconds to seconds, for cache reuse, for search optimization purpose
			//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
			//==>
			//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:00.000Z","format":"strict_date_optional_time"}
			body := ctx.Request.Body()
			precisionLimit,_:=this.GetInt("time_precision",4)
			//0-9: 时分秒微妙 00:00:00:000
			if precisionLimit>9{
				precisionLimit=9
			}
			if precisionLimit<0{
				precisionLimit=0
			}

			//TODO get time field from index pattern settings
			ok := util.ProcessJsonData(&body, []byte("range"), []byte("gte"),false,[]byte("gte"), []byte("}"),128 , func(data []byte,start, end int) {
				startProcess := false
				precisionOffset := 0
				matchCount:=0
				for i, v := range body[start:end] {
					if v == 84 { //T
						startProcess = true
						precisionOffset = 0
						matchCount++
						continue
					}
					if startProcess && v > 47 && v < 58 {
						precisionOffset++
						if precisionOffset <= precisionLimit {
							continue
						} else if precisionOffset > 9 {
							startProcess = false
							continue
						}
						if matchCount==1{
							body[start+i] = 48
						}else if matchCount==2{
							//prev,_:=strconv.Atoi(string(body[start+i-1]))
							prev:=body[start+i-1]

							if precisionOffset==1{
								body[start+i] = 50
								continue
							}

							if precisionOffset==2{
								if prev==48{//int:0
									body[start+i] = 57
									continue
								}
								if prev==49{ //int:1
									body[start+i] = 57
									continue
								}
								if prev==50{ //int:2
									body[start+i] = 51
									continue
								}
							}
							if precisionOffset==3{
								body[start+i] = 53
								continue
							}
							if precisionOffset==4{
								if prev!=54{//int:6
									body[start+i] = 57
									continue
								}
							}
							if precisionOffset==5{
								body[start+i] = 53
								continue
							}
							if precisionOffset>=6{
								body[start+i] = 57
								continue
							}

						}

					}

				}
			})
			if ok {
				ctx.Request.SetBody(body)
			}
			//{"size":0,"query":{"bool":{"must":[{"range":{"@timestamp":{"gte":"2019-09-26T15:16:59.127Z","lte":"2020-09-26T15:16:59.127Z","format":"strict_date_optional_time"}}}],"filter":[{"match_all":{}}],"should":[],"must_not":[]}},"aggs":{"61ca57f1-469d-11e7-af02-69e470af7417":{"terms":{"field":"log.file.path","order":{"_count":"desc"}},"aggs":{"timeseries":{"date_histogram":{"field":"@timestamp","min_doc_count":0,"time_zone":"Asia/Shanghai","extended_bounds":{"min":1569511019127,"max":1601133419127},"fixed_interval":"86400s"},"aggs":{"61ca57f2-469d-11e7-af02-69e470af7417":{"bucket_script":{"buckets_path":{"count":"_count"},"script":{"source":"count * 1","lang":"expression"},"gap_policy":"skip"}}}}},"meta":{"timeField":"@timestamp","intervalString":"86400s","bucketSize":86400,"seriesId":"61ca57f1-469d-11e7-af02-69e470af7417"}}},"timeout":"30000ms"}
		}
}
