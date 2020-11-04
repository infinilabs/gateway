package filter

import (
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/output/logging"
)

func Init()  {
	requestLogging:= logging.RequestLogging{}
	//flow:= common.NewFilterFlow(requestLogging.Name(),requestLogging.Process)
	common.RegisterFilter(requestLogging)
}
