/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
)

func ValidateBulkRequest(where, body string) {
	stringLines := strings.Split(body, "\n")
	if len(stringLines)==0{
		log.Error("invalid json lines, empty")
		return
	}
	obj := map[string]interface{}{}
	for _, v := range stringLines {
		err := util.FromJSONBytes([]byte(v), &obj)
		if err != nil {
			log.Error("invalid json,", where, ",", util.SubString(v, 0, 512), err)
			break
		}
	}
}
