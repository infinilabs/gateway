package filter

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/output/logging"
	"infini.sh/gateway/proxy/output/stdout"
)

func Init()  {
	//flow:= common.NewFilterFlow(requestLogging.Name(),requestLogging.Process)
	common.RegisterFilter(logging.RequestLogging{})
	common.RegisterFilter(stdout.EchoDot{})
}
