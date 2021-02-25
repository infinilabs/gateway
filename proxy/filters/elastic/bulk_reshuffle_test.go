package elastic

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"infini.sh/framework/core/util"
	"testing"
)

func TestInsertIDMeta(t *testing.T) {
	data:=[]byte("{\"index\":{\"_index\":\"test\",\"_type\":\"doc\"}}")

	//{"index":{"_index":"test"
	//	,"_type":"doc"}}

	id:="myid"
	newData:=util.InsertBytesAfterField(&data,[]byte("\"_index\""),[]byte("\""),[]byte("\""),[]byte(", \"_id\":\""+id+"\""))
	fmt.Println(string(newData))


	assert.Equal(t,newData,[]byte("{\"index\":{\"_index\":\"test\", \"_id\":\"myid\",\"_type\":\"doc\"}}"))

	newData,id=insertUUID(data)
	fmt.Println(string(newData),id)

	data=[]byte("{\"index\":{\"_type\":\"doc\",\"_index\":\"test\"}}")
	newData,id=insertUUID(data)
	fmt.Println(string(newData),id)


}
func TestParseActionMeta(t *testing.T) {

	data := []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")


	//for i,v:=range data{
	//	fmt.Println(i,string(v))
	//}

	action, indexb, idb := parseActionMeta(data)
	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")


	data = []byte("{\"index\":{\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\",\"_index\":\"medcl1\"}}")

	action, indexb, idb = parseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")

	data=[]byte("{\"index\":{\"_index\":\"test\",\"_type\":\"doc\"}}")
	action, indexb, idb = parseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )

}

