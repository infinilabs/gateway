package date_range_precision_tuning

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"testing"
)

func TestDatePrecisionTuning(t *testing.T) {
	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.152Z\",\"lte\":\"2020-09-26T08:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")
	ctx.Request.SetBody(data)

	filter.config.TimePrecision=0
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T23:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=1
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T09:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=2
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:00:00.000Z\",\"lte\":\"2020-09-26T08:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=3
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:20:00.000Z\",\"lte\":\"2020-09-26T08:29:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=4
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:00.000Z\",\"lte\":\"2020-09-26T08:21:59.999Z\",\"format\":\"strict_date_optional_time\"}")


	ctx.Request.SetBody(data)
	filter.config.TimePrecision=5
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:10.000Z\",\"lte\":\"2020-09-26T08:21:19.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=6
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.000Z\",\"lte\":\"2020-09-26T08:21:12.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=7
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.100Z\",\"lte\":\"2020-09-26T08:21:12.199Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=8
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.150Z\",\"lte\":\"2020-09-26T08:21:12.159Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=9
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.152Z\",\"lte\":\"2020-09-26T08:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")

}

func TestDatePrecisionTuning1(t *testing.T) {
	filter:= DatePrecisionTuning{config: &defaultConfig}

	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T22:21:12.152Z\",\"lte\":\"2020-09-26T22:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")
	fmt.Println(string(data))

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=0
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T23:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")


	ctx.Request.SetBody(data)
	filter.config.TimePrecision=9
	filter.Filter(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T22:21:12.152Z\",\"lte\":\"2020-09-26T22:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")

}

func TestDatePrecisionTuning2(t *testing.T) {

	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"query\":{\"bool\":{\"filter\":[{\"term\":{\"type\":\"node_stats\"}},{\"term\":{\"cluster_uuid\":\"OT_m4gvgTvqb-LZjU66NLg\"}},{\"terms\":{\"source_node.uuid\":[\"qIgTsxtuQ8mzAGiBATkqHw\"]}},{\"range\":{\"timestamp\":{\"format\":\"epoch_millis\",\"gte\":1612315985132,\"lte\":1612319585132}}}]}},\"aggs\":{\"nodes\":{\"terms\":{\"field\":\"source_node.uuid\",\"include\":[\"qIgTsxtuQ8mzAGiBATkqHw\"],\"size\":10000},\"aggs\":{\"by_date\":{\"date_histogram\":{\"field\":\"timestamp\",\"min_doc_count\":0,\"fixed_interval\":\"30s\"},\"aggs\":{\"odh_node_cgroup_quota__usage\":{\"max\":{\"field\":\"node_stats.os.cgroup.cpuacct.usage_nanos\"}},\"odh_node_cgroup_quota__periods\":{\"max\":{\"field\":\"node_stats.os.cgroup.cpu.stat.number_of_elapsed_periods\"}},\"odh_node_cgroup_quota__quota\":{\"min\":{\"field\":\"node_stats.os.cgroup.cpu.cfs_quota_micros\"}},\"odh_node_cgroup_quota__usage_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_cgroup_quota__usage\",\"gap_policy\":\"skip\",\"unit\":\"1s\"}},\"odh_node_cgroup_quota__periods_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_cgroup_quota__periods\",\"gap_policy\":\"skip\",\"unit\":\"1s\"}},\"odh_node_cgroup_throttled__metric\":{\"max\":{\"field\":\"node_stats.os.cgroup.cpu.stat.time_throttled_nanos\"}},\"odh_node_cgroup_throttled__metric_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_cgroup_throttled__metric\",\"unit\":\"1s\"}},\"odh_node_cpu_utilization__metric\":{\"max\":{\"field\":\"node_stats.process.cpu.percent\"}},\"odh_node_cpu_utilization__metric_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_cpu_utilization__metric\",\"unit\":\"1s\"}},\"odh_node_load_average__metric\":{\"max\":{\"field\":\"node_stats.os.cpu.load_average.1m\"}},\"odh_node_load_average__metric_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_load_average__metric\",\"unit\":\"1s\"}},\"odh_node_jvm_mem_percent__metric\":{\"max\":{\"field\":\"node_stats.jvm.mem.heap_used_percent\"}},\"odh_node_jvm_mem_percent__metric_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_jvm_mem_percent__metric\",\"unit\":\"1s\"}},\"odh_node_free_space__metric\":{\"max\":{\"field\":\"node_stats.fs.total.available_in_bytes\"}},\"odh_node_free_space__metric_deriv\":{\"derivative\":{\"buckets_path\":\"odh_node_free_space__metric\",\"unit\":\"1s\"}}}}}}}}")
	fmt.Println(string(data))

	precisionLimit:=4
	ok := util.ProcessJsonData(&data,  []byte("range"),150,[][]byte{[]byte("gte"),[]byte("lte")},false, []byte("gte"),[]byte("}"),128, func(data []byte,start, end int) {
		fmt.Println(string(data))
		startProcess := false
		precisionOffset := 0
		matchCount:=0
		for i, v := range data[start:end] {
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
					data[start+i] = 48
				}else if matchCount==2{
					//prev,_:=strconv.Atoi(string(body[start+i-1]))
					prev:=data[start+i-1]

					if precisionOffset==1{
						data[start+i] = 50
						continue
					}

					if precisionOffset==2{
						if prev==48{//int:0
							data[start+i] = 57
							continue
						}
						if prev==49{ //int:1
							data[start+i] = 57
							continue
						}
						if prev==50{ //int:2
							data[start+i] = 51
							continue
						}
					}
					if precisionOffset==3{
						data[start+i] = 53
						continue
					}
					if precisionOffset==4{
						if prev!=54{//int:6
							data[start+i] = 57
							continue
						}
					}
					if precisionOffset==5{
						data[start+i] = 53
						continue
					}
					if precisionOffset>=6{
						data[start+i] = 57
						continue
					}

				}

			}

		}
	})

	fmt.Println(ok)

	ctx.Request.SetBody(data)
	filter.config.TimePrecision=4
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	//assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T23:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

}

func TestDatePrecisionTuning3(t *testing.T) {

	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\n  \"query\": {\n    \"query_string\": {\n      \"default_field\": \"title\",\n      \"query\": \"this range AND gte TO goodbye 2019-09-26T00:10:00.000Z thus\"\n    }\n  }\n}")
	ctx.Request.SetBody(data)

	filter.config.TimePrecision=0
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\n  \"query\": {\n    \"query_string\": {\n      \"default_field\": \"title\",\n      \"query\": \"this range AND gte TO goodbye 2019-09-26T00:10:00.000Z thus\"\n    }\n  }\n}")

}
