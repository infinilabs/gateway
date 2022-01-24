/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package transform

import (
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

type RequestBodyJsonSet struct {
	IgnoreMissing bool     `config:"ignore_missing"`
	Path          []string `config:"path"`
	m             map[string]string
}

func (filter *RequestBodyJsonSet) Name() string {
	return "request_body_json_set"
}

func (filter *RequestBodyJsonSet) Filter(ctx *fasthttp.RequestCtx) {

	bodyBytes := ctx.Request.GetRawBody()

	//var err error
	if len(filter.m) > 0 {
		if len(bodyBytes) == 0 {
			bodyBytes = []byte("{}")
		}

		for path, value := range filter.m {
			pathArray := strings.Split(path, ".")
			v, t, offset, err := jsonparser.Get(bodyBytes, pathArray...)
			if t == jsonparser.NotExist && filter.IgnoreMissing {
				log.Debugf("path:%v, value:%v, %v, %v, %v, %v", path, value, err, v, t, offset)
				continue
			}

			bodyBytes, err = jsonparser.Set(bodyBytes, []byte(value), pathArray...)
			if err != nil {
				log.Errorf("path:%v, value:%v, %v", path, value, err)
				return
			}
		}
		ctx.Request.SetRawBody(bodyBytes)
		return
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_body_json_set",NewRequestBodyJsonSet,&RequestBodyJsonSet{})
}

func NewRequestBodyJsonSet(c *config.Config) (pipeline.Filter, error) {

	runner := RequestBodyJsonSet{
		IgnoreMissing: false,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.m = map[string]string{}
	for _, item := range runner.Path {
		k, v, err := util.ConvertStringToMap(item, "->")
		if err != nil {
			panic(err)
		}
		runner.m[k] = v
	}

	return &runner, nil
}
