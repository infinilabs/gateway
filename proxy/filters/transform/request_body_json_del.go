/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

type RequestBodyJsonDel struct {
	IgnoreMissing bool     `config:"ignore_missing"`
	Path          []string `config:"path"`
}

func (filter *RequestBodyJsonDel) Name() string {
	return "request_body_json_del"
}

func (filter *RequestBodyJsonDel) Filter(ctx *fasthttp.RequestCtx) {
	bodyBytes := ctx.Request.GetRawBody()
	if len(filter.Path) > 0 {
		if len(bodyBytes) == 0 {
			bodyBytes = []byte("{}")
		}

		for _, path := range filter.Path {
			pathArray := strings.Split(path, ".")
			v, t, offset, err := jsonparser.Get(bodyBytes, pathArray...)
			if t == jsonparser.NotExist && filter.IgnoreMissing {
				log.Debugf("path:%v, %v, %v, %v, %v", path, err, v, t, offset)
				continue
			}

			bodyBytes = jsonparser.Delete(bodyBytes, pathArray...)
			if err != nil {
				log.Errorf("path:%v, %v", path, err)
				return
			}
		}
		ctx.Request.SetRawBody(bodyBytes)
		return
	}
}

func init() {
	pipeline.RegisterFilterPlugin("request_body_json_del",NewRequestBodyJsonDel)
}

func NewRequestBodyJsonDel(c *config.Config) (pipeline.Filter, error) {

	runner := RequestBodyJsonDel{
		IgnoreMissing: false,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
