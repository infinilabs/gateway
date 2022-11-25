/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package echo

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type Echo struct {
	RepeatTimes int    `config:"repeat"    type:"number"  default_value:"1" `
	Continue    bool   `config:"continue"  type:"bool"    default_value:"true" `
	Terminal    bool   `config:"stdout"    type:"bool"    default_value:"false" `
	Response    bool   `config:"response"    type:"bool"    default_value:"true" `
	Message     string `config:"message"   type:"string"  default_value:"." `
	Messages    []string `config:"messages" `
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("echo", New, &Echo{})
}

func New(c *config.Config) (pipeline.Filter, error) {

	runner := Echo{
		Response: true,
		RepeatTimes: 1,
		Continue:    true,
		Message:     ".",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *Echo) Name() string {
	return "echo"
}

func (filter *Echo) Filter(ctx *fasthttp.RequestCtx) {
	str := filter.Message
	size := filter.RepeatTimes
	for i := 0; i < size; i++ {
		if filter.Response{
			ctx.WriteString(str)
		}
		if filter.Terminal {
			fmt.Print(str)
		}
	}
	if !filter.Continue {
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
	}
}
