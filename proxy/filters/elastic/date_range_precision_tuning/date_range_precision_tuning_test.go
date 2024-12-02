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

func TestDatePrecisionTuning4(t *testing.T) {
	//data:=[]byte("{\"size\":0,\"query\":{\"bool\":{\"must\":[{\"range\":{\"@timestamp\":{\"gte\":\"2021-12-29T08:50:33.345Z\",\"lte\":\"2022-01-05T08:50:33.345Z\",\"format\":\"strict_date_optional_time\"}}},{\"bool\":{\"must\":[],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}}],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}},\"aggs\":{\"timeseries\":{\"date_histogram\":{\"field\":\"@timestamp\",\"min_doc_count\":0,\"time_zone\":\"Asia/Shanghai\",\"extended_bounds\":{\"min\":1640767833345,\"max\":1641372633345},\"calendar_interval\":\"1d\"},\"aggs\":{\"61ca57f2-469d-11e7-af02-69e470af7417\":{\"cardinality\":{\"field\":\"source.ip\"}}},\"meta\":{\"timeField\":\"@timestamp\",\"intervalString\":\"1d\",\"bucketSize\":86400,\"seriesId\":\"61ca57f1-469d-11e7-af02-69e470af7417\"}}},\"timeout\":\"30000ms\"}")
	data:=[]byte("12345121231231231232131312312312{\"range\":{\"@timestamp\":{\"gte\":\"2021-12-29T08:50:33.345Z\",\"lte\":\"2022-01-05T08:50:33.345Z\",\"format\":\"strict_date_optional_time\"}}},{\"bool\":{\"must\":[],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}}],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}},\"aggs\":{\"timeseries\":{\"date_histogram\":{\"field\":\"@timestamp\",\"min_doc_count\":0,\"time_zone\":\"Asia/Shanghai\",\"extended_bounds\":{\"min\":1640767833345,\"max\":1641372633345},\"calendar_interval\":\"1d\"},\"aggs\":{\"61ca57f2-469d-11e7-af02-69e470af7417\":{\"cardinality\":{\"field\":\"source.ip\"}}},\"meta\":{\"timeField\":\"@timestamp\",\"intervalString\":\"1d\",\"bucketSize\":86400,\"seriesId\":\"61ca57f1-469d-11e7-af02-69e470af7417\"}}},\"timeout\":\"30000ms\"}")
	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBody(data)

	filter.config.TimePrecision=2
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"12345121231231231232131312312312{\"range\":{\"@timestamp\":{\"gte\":\"2021-12-29T08:00:00.000Z\",\"lte\":\"2022-01-05T08:59:59.999Z\",\"format\":\"strict_date_optional_time\"}}},{\"bool\":{\"must\":[],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}}],\"filter\":[{\"match_all\":{}}],\"should\":[],\"must_not\":[]}},\"aggs\":{\"timeseries\":{\"date_histogram\":{\"field\":\"@timestamp\",\"min_doc_count\":0,\"time_zone\":\"Asia/Shanghai\",\"extended_bounds\":{\"min\":1640767833345,\"max\":1641372633345},\"calendar_interval\":\"1d\"},\"aggs\":{\"61ca57f2-469d-11e7-af02-69e470af7417\":{\"cardinality\":{\"field\":\"source.ip\"}}},\"meta\":{\"timeField\":\"@timestamp\",\"intervalString\":\"1d\",\"bucketSize\":86400,\"seriesId\":\"61ca57f1-469d-11e7-af02-69e470af7417\"}}},\"timeout\":\"30000ms\"}")

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

func TestDatePrecisionTuning5(t *testing.T) {

	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"version\":true,\"size\":500,\"sort\":[{\"createTime\":{\"order\":\"desc\",\"unmapped_type\":\"boolean\"}}],\"aggs\":{\"2\":{\"date_histogram\":{\"field\":\"createTime\",\"fixed_interval\":\"30s\",\"time_zone\":\"Asia/Shanghai\",\"min_doc_count\":1}}},\"stored_fields\":[\"*\"],\"script_fields\":{},\"docvalue_fields\":[{\"field\":\"createTime\",\"format\":\"date_time\"}],\"_source\":{\"excludes\":[]},\"query\":{\"bool\":{\"must\":[],\"filter\":[{\"match_all\":{}},{\"range\":{\"createTime\":{\"gte\":\"2022-08-15T11:26:51.953Z\",\"lte\":\"2022-08-15T11:41:51.953Z\",\"format\":\"strict_date_optional_time\"}}}],\"should\":[],\"must_not\":[]}},\"highlight\":{\"pre_tags\":[\"@kibana-highlighted-field@\"],\"post_tags\":[\"@/kibana-highlighted-field@\"],\"fields\":{\"*\":{}},\"fragment_size\":2147483647}}")
	ctx.Request.SetBody(data)

	filter.config.TimePrecision=6
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"version\":true,\"size\":500,\"sort\":[{\"createTime\":{\"order\":\"desc\",\"unmapped_type\":\"boolean\"}}],\"aggs\":{\"2\":{\"date_histogram\":{\"field\":\"createTime\",\"fixed_interval\":\"30s\",\"time_zone\":\"Asia/Shanghai\",\"min_doc_count\":1}}},\"stored_fields\":[\"*\"],\"script_fields\":{},\"docvalue_fields\":[{\"field\":\"createTime\",\"format\":\"date_time\"}],\"_source\":{\"excludes\":[]},\"query\":{\"bool\":{\"must\":[],\"filter\":[{\"match_all\":{}},{\"range\":{\"createTime\":{\"gte\":\"2022-08-15T11:26:51.000Z\",\"lte\":\"2022-08-15T11:41:51.999Z\",\"format\":\"strict_date_optional_time\"}}}],\"should\":[],\"must_not\":[]}},\"highlight\":{\"pre_tags\":[\"@kibana-highlighted-field@\"],\"post_tags\":[\"@/kibana-highlighted-field@\"],\"fields\":{\"*\":{}},\"fragment_size\":2147483647}}")

}


func TestDatePrecisionTuning6(t *testing.T) {

	filter:= DatePrecisionTuning{config: &defaultConfig}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.SetRequestURI("/_search")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"size\":0,\"query\":{\"bool\":{\"filter\":[{\"range\":{\"@timestamp\":{\"gte\":\"2024-04-24T08:23:13.301Z\",\"lte\":\"2024-04-24T09:21:12.152Z\",\"format\":\"strict_date_optional_time\"}}},{\"query_string\":{\"analyze_wildcard\":true,\"query\":\"*\"}}]}},\"aggs\":{\"2\":{\"date_histogram\":{\"interval\":\"200ms\",\"field\":\"@timestamp\",\"min_doc_count\":0,\"format\":\"epoch_millis\"},\"aggs\":{}}}}")
	ctx.Request.SetBody(data)

	filter.config.TimePrecision=4
	filter.Filter(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"size\":0,\"query\":{\"bool\":{\"filter\":[{\"range\":{\"@timestamp\":{\"gte\":\"2024-04-24T08:23:00.000Z\",\"lte\":\"2024-04-24T09:21:59.999Z\",\"format\":\"strict_date_optional_time\"}}},{\"query_string\":{\"analyze_wildcard\":true,\"query\":\"*\"}}]}},\"aggs\":{\"2\":{\"date_histogram\":{\"interval\":\"200ms\",\"field\":\"@timestamp\",\"min_doc_count\":0,\"format\":\"epoch_millis\"},\"aggs\":{}}}}")

}

