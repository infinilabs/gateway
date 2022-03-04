package common

import "github.com/buger/jsonparser"

func GetScrollHitsTotal(version int, data []byte) (int64, error) {
	if version >= 7 {
		return jsonparser.GetInt(data, "hits", "total", "value")
	} else {
		return jsonparser.GetInt(data, "hits", "total")
	}
}
