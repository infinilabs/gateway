package elastic

import (
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestDatePrecisionTuning(t *testing.T) {
	filter:=DatePrecisionTuning{}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.152Z\",\"lte\":\"2020-09-26T08:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")
	ctx.Request.SetBody(data)

	filter.Set("time_precision",0)
	filter.Process(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T23:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",1)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T09:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",2)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:00:00.000Z\",\"lte\":\"2020-09-26T08:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",3)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:20:00.000Z\",\"lte\":\"2020-09-26T08:29:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",4)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:00.000Z\",\"lte\":\"2020-09-26T08:21:59.999Z\",\"format\":\"strict_date_optional_time\"}")


	ctx.Request.SetBody(data)
	filter.Set("time_precision",5)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:10.000Z\",\"lte\":\"2020-09-26T08:21:19.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",6)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.000Z\",\"lte\":\"2020-09-26T08:21:12.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",7)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.100Z\",\"lte\":\"2020-09-26T08:21:12.199Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",8)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.150Z\",\"lte\":\"2020-09-26T08:21:12.159Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",9)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T08:21:12.152Z\",\"lte\":\"2020-09-26T08:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")

}


func TestDatePrecisionTuning1(t *testing.T) {
	filter:=DatePrecisionTuning{}
	ctx:=&fasthttp.RequestCtx{}
	ctx.Request=fasthttp.Request{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)

	data:=[]byte("{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T22:21:12.152Z\",\"lte\":\"2020-09-26T22:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")
	fmt.Println(string(data))

	ctx.Request.SetBody(data)
	filter.Set("time_precision",0)
	filter.Process(ctx)
	rePrecisedBody:=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T00:00:00.000Z\",\"lte\":\"2020-09-26T23:59:59.999Z\",\"format\":\"strict_date_optional_time\"}")

	ctx.Request.SetBody(data)
	filter.Set("time_precision",9)
	filter.Process(ctx)
	rePrecisedBody=string(ctx.Request.Body())
	fmt.Println(rePrecisedBody)
	assert.Equal(t,rePrecisedBody,"{\"range\":{\"@timestamp\":{\"gte\":\"2019-09-26T22:21:12.152Z\",\"lte\":\"2020-09-26T22:21:12.152Z\",\"format\":\"strict_date_optional_time\"}")

}