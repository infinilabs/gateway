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
	config *Config
}

type Config struct {
	RepeatTimes int    `config:"repeat"`
	Continue    bool   `config:"continue"`
	Terminal    bool   `config:"stdout"`
	Message     string `config:"message"`
}

func New(c *config.Config) (pipeline.Filter, error) {

	cfg := Config{
		RepeatTimes: 1,
		Continue:    true,
		Message:     ".",
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner := EchoMessage{config: &cfg}

	return &runner, nil
}

func (filter *EchoMessage) Name() string {
	return "echo"
}

func (filter *EchoMessage) Filter(ctx *fasthttp.RequestCtx) {
	str := filter.config.Message
	size := filter.config.RepeatTimes
	for i := 0; i < size; i++ {
		ctx.WriteString(str)
		if filter.config.Terminal {
			fmt.Print(str)
		}
	}
	if !filter.config.Continue {
		ctx.Finished()
	}
}


