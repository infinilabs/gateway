package filter

import (
	"infini.sh/gateway/modules/proxy/common"
	"infini.sh/gateway/modules/proxy/filter/request_logging"
)

func Init()  {
	requestLogging:=request_logging.RequestLogging{}
	flow:=common.NewFilterFlow("request_logging",requestLogging.Process)
	common.RegisterFlow(flow)

}
