/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package bulk_indexing

import (
	"github.com/valyala/fasttemplate"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"io"
)

type BulkContext struct {
	pipelineContext *pipeline.Context
	TaskContext *util.MapStr
	ElasticsearchContext *elastic.ElasticsearchMetadata
}

func (ctx *BulkContext)GetValue(k string) (interface{}, error){

	if util.ContainStr(k,"$[["){
		template, err := fasttemplate.NewTemplate(k, "$[[", "]]")
		if err != nil {
			panic(err)
		}
		k,err = template.ExecuteFuncStringWithErr(func(w io.Writer, tag string) (int, error) {
			variable,err := ctx.GetValue(tag)
			if err!=nil{
				return 0,err
			}
			return w.Write([]byte(util.ToString(variable)))
		})
		if err==nil{
			return ctx.GetValue(k)
		}
	}

	if ctx.TaskContext!=nil{
		v,err:=ctx.TaskContext.GetValue(k)
		if err==nil{
			return v,err
		}
	}

	if ctx.ElasticsearchContext!=nil{
		v,err:=ctx.ElasticsearchContext.GetValue(k)
		if err==nil{
			return v,err
		}
	}

	return ctx.pipelineContext.GetValue(k)
}

