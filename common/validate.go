/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"strings"
)

func ValidateBulkRequest(where, body string) {
	stringLines := strings.Split(body, "\n")
	for _, v := range stringLines {
		obj := map[string]interface{}{}
		err := util.FromJSONBytes([]byte(v), &obj)
		if err != nil {
			seelog.Error("invalid json,", where, ",", util.SubString(v, 0, 512), err)
			break
		}
	}
}

