/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package echo

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type EchoMessage struct {
	RepeatTimes int    `config:"repeat"`
	Continue    bool   `config:"continue"`
	Terminal    bool   `config:"stdout"`
	Message     string `config:"message"`
}

func init() {
	pipeline.RegisterFilterPlugin("echo", New)
}

func New(c *config.Config) (pipeline.Filter, error) {

	runner := EchoMessage{
		RepeatTimes: 1,
		Continue:    true,
		Message:     ".",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}

func (filter *EchoMessage) Name() string {
	return "echo"
}

func (filter *EchoMessage) Filter(ctx *fasthttp.RequestCtx) {
	str := filter.Message
	size := filter.RepeatTimes
	for i := 0; i < size; i++ {
		ctx.WriteString(str)
		if filter.Terminal {
			fmt.Print(str)
		}
	}
	if !filter.Continue {
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
	}
}
