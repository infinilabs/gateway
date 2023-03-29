package common

import (
	"github.com/buger/jsonparser"

	"infini.sh/framework/core/elastic"
)

func GetScrollHitsTotal(version elastic.Version, data []byte) (int64, error) {
	if version.Distribution == elastic.Elasticsearch && version.Major < 7 {
		return jsonparser.GetInt(data, "hits", "total")
	}
	return jsonparser.GetInt(data, "hits", "total", "value")
}
