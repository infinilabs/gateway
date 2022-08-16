package date_range_precision_tuning

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type DatePrecisionTuning struct {
	config *Config
}

type Config struct {
	PathKeywords  []string `config:"path_keywords"`
	TimePrecision int      `config:"time_precision"`
}

var defaultConfig = Config{
	PathKeywords:  builtinKeywords,
	TimePrecision: 4,
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("date_range_precision_tuning", New,&defaultConfig)
}

func New(c *config.Config) (pipeline.Filter, error) {
	cfg := defaultConfig
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if cfg.TimePrecision > 9 {
		cfg.TimePrecision = 9
	}

	if cfg.TimePrecision < 0 {
		cfg.TimePrecision = 0
	}

	runner := DatePrecisionTuning{config: &cfg}

	return &runner, nil
}

func (this *DatePrecisionTuning) Name() string {
	return "date_range_precision_tuning"
}

var builtinKeywords = []string{"_search", "_async_search"}

func (this *DatePrecisionTuning) Filter(ctx *fasthttp.RequestCtx) {

	if ctx.Request.GetBodyLength() <= 0 {
		return
	}

	path := string(ctx.RequestURI())

	if util.ContainsAnyInArray(path, this.config.PathKeywords) {
		//request normalization
		//timestamp precision processing, scale time from million seconds to seconds, for cache reuse, for search optimization purpose
		//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
		//==>
		//{"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:00.000Z","format":"strict_date_optional_time"}
		body := ctx.Request.GetRawBody()
		//0-9: 时分秒微妙 00:00:00:000

		//TODO get time field from index pattern settings
		ok := util.ProcessJsonData(&body, []byte("range"), 150, [][]byte{[]byte("gte"), []byte("lte")}, false, []byte("gte"), []byte("}"), 128, func(data []byte, start, end int) {
			startProcess := false
			precisionOffset := 0
			matchCount := 0
			//fmt.Println("body[start:end]: ",string(body[start:end]))
			block:=body[start:end]
			len:=len(block)-1
			for i, v := range block {
				if i>1 &&i <len{
					//fmt.Println(i,",",string(v),",",block[i-1],",",block[i+1])
					left:=block[i-1]
					right:=block[i+1]
					if v == 84 &&left > 47 && left < 58 &&right > 47 && right < 58{ //T
						//fmt.Println("star process")
						startProcess = true
						precisionOffset = 0
						matchCount++
						continue
					}
				}


				if startProcess && v > 47 && v < 58 {
					precisionOffset++
					if precisionOffset <= this.config.TimePrecision {
						continue
					} else if precisionOffset > 9 {
						startProcess = false
						continue
					}
					if matchCount == 1 {
						body[start+i] = 48
					} else if matchCount == 2 {
						//prev,_:=strconv.Atoi(string(body[start+i-1]))
						prev := body[start+i-1]

						if precisionOffset == 1 {
							body[start+i] = 50
							continue
						}

						if precisionOffset == 2 {
							if prev == 48 { //int:0
								body[start+i] = 57
								continue
							}
							if prev == 49 { //int:1
								body[start+i] = 57
								continue
							}
							if prev == 50 { //int:2
								body[start+i] = 51
								continue
							}
						}
						if precisionOffset == 3 {
							body[start+i] = 53
							continue
						}
						if precisionOffset == 4 {
							if prev != 54 { //int:6
								body[start+i] = 57
								continue
							}
						}
						if precisionOffset == 5 {
							body[start+i] = 53
							continue
						}
						if precisionOffset >= 6 {
							body[start+i] = 57
							continue
						}

					}

				}

			}
		})

		if ok {
			ctx.Request.SetRawBody(body)
		}
		//{"size":0,"query":{"bool":{"must":[{"range":{"@timestamp":{"gte":"2019-09-26T15:16:59.127Z","lte":"2020-09-26T15:16:59.127Z","format":"strict_date_optional_time"}}}],"filter":[{"match_all":{}}],"should":[],"must_not":[]}},"aggs":{"61ca57f1-469d-11e7-af02-69e470af7417":{"terms":{"field":"log.file.path","order":{"_count":"desc"}},"aggs":{"timeseries":{"date_histogram":{"field":"@timestamp","min_doc_count":0,"time_zone":"Asia/Shanghai","extended_bounds":{"min":1569511019127,"max":1601133419127},"fixed_interval":"86400s"},"aggs":{"61ca57f2-469d-11e7-af02-69e470af7417":{"bucket_script":{"buckets_path":{"count":"_count"},"script":{"source":"count * 1","lang":"expression"},"gap_policy":"skip"}}}}},"meta":{"timeField":"@timestamp","intervalString":"86400s","bucketSize":86400,"seriesId":"61ca57f1-469d-11e7-af02-69e470af7417"}}},"timeout":"30000ms"}
	}
}
