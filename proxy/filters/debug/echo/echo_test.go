package echo

import (
	"fmt"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"testing"
)

func TestExtractFieldWithTags(t *testing.T) {

	echo := &Echo{}
	results := util.GetFieldAndTags(echo, []string{"config", "type", "sub_type", "default_value"})
	fmt.Println(string(util.MustToJSONBytes(results)))
	results1 := pipeline.ExtractFilterMetadata(echo)
	fmt.Println(string(util.MustToJSONBytes(results1)))


}