package elastic

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestParseActionMeta(t *testing.T) {

	data := []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")

	action, indexb, idb := parseActionMeta(data)
	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")
}
