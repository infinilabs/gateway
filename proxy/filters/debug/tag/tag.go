/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package echo

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type Tag struct {
	AddTags       []string `config:"add" `
	RemoveTags    []string `config:"remove" `
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("tag", New, &Tag{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	runner := Tag{
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *Tag) Name() string {
	return "tag"
}


func (filter *Tag) Filter(ctx *fasthttp.RequestCtx) {

		if len(filter.AddTags)>0||len(filter.RemoveTags)>0{
			ctx.UpdateTags(filter.AddTags,filter.RemoveTags)
		}

}
