package elastic

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/magiconair/properties/assert"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
	"strings"
	"testing"
)

//func TestInsertIDMeta(t *testing.T) {
//	data:=[]byte("{\"index\":{\"_index\":\"test\",\"_type\":\"doc\"}}")
//
//	//{"index":{"_index":"test"
//	//	,"_type":"doc"}}
//
//	id:="myid"
//	newData:=util.InsertBytesAfterField(&data,[]byte("\"_index\""),[]byte("\""),[]byte("\""),[]byte(", \"_id\":\""+id+"\""))
//	fmt.Println(string(newData))
//
//
//	assert.Equal(t,newData,[]byte("{\"index\":{\"_index\":\"test\", \"_id\":\"myid\",\"_type\":\"doc\"}}"))
//
//	newData,id=insertUUID(data)
//	fmt.Println(string(newData),id)
//
//	data=[]byte("{\"index\":{\"_type\":\"doc\",\"_index\":\"test\"}}")
//	newData,id=insertUUID(data)
//	fmt.Println(string(newData),id)
//
//
//}

func TestParseActionMeta1(t *testing.T) {

	data := []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")
	action := util.ExtractFieldFromBytes(&data, elastic.ActionStart, elastic.ActionEnd, nil)
	fmt.Println(string(action))
	indexb,_,_,_:=jsonparser.Get(data,util.UnsafeBytesToString(action),"_index")
	fmt.Println(string(indexb))
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	idb,_,_,_:=jsonparser.Get(data,util.UnsafeBytesToString(action),"_id")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")

	//update json bytes
	new,_:=jsonparser.Set(data, []byte("medcl2"),"index","_index")
	fmt.Println("new:",string(new))

}

func TestParseActionMeta2(t *testing.T) {

	data := []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")

	action, indexb, typeb, idb,_ := elastic.ParseActionMeta(data)
	fmt.Println(string(action), string(indexb), string(idb))
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")


	data = []byte("{\"index\":{\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\",\"_index\":\"medcl1\"}}")

	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")


	data = []byte("{\"index\":{\"_id\":\"GZq-bnYBC53QmW9Kk2ve\",\"_type\":\"_doc\",\"_index\":\"medcl1\"}}")

	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")

	data = []byte("{\"index\":{\"_index\":\"test\",\"_type\":\"doc\"}}")
	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{\"delete\":{\"_index\":\"test\",\"_type\":\"_doc\"}}")
	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"delete")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{\"create\":{\"_index\":\"test\",\"_type\":\"_doc\"}}")
	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"create")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{ \"update\" : {\"_id\" : \"1\", \"_index\" : \"test\"} }")
	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"update")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"")
	assert.Equal(t,string(idb),"1")

	data = []byte("{ \"update\" : {\"_index\" : \"test\"} }")
	action, indexb, typeb, idb,_ = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"update")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"")
	assert.Equal(t,string(idb),"")


}

func TestParseActionMeta3(t *testing.T) {

	data := []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")
	newData,err := updateJsonWithNewIndex("index",data,"newIndex","newType","newId")
	fmt.Println(err,string(newData))
	assert.Equal(t,string(newData),"{\"index\":{\"_index\":\"newIndex\",\"_type\":\"newType\",\"_id\":\"newId\"}}")


	data = []byte("{\"index\":{\"_index\":\"medcl1\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\"}}")
	newData,err = updateJsonWithNewIndex("index",data,"newIndex","newType","newId")
	fmt.Println(err,string(newData))
	assert.Equal(t,string(newData),"{\"index\":{\"_index\":\"newIndex\",\"_id\":\"newId\",\"_type\":\"newType\"}}")

	data = []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"doc1\"}}")
	newData,err = updateJsonWithNewIndex("index",data,"newIndex","newType","newId")
	fmt.Println(err,string(newData))
	assert.Equal(t,string(newData),"{\"index\":{\"_index\":\"newIndex\",\"_type\":\"newType\",\"_id\":\"newId\"}}")



	data = []byte("{\"index\":{\"_index\":\"medcl1\",\"_type\":\"doc1\"}}")
	newData,err = updateJsonWithNewIndex("index",data,"","","newId")
	fmt.Println(err,string(newData))
	assert.Equal(t,string(newData),"{\"index\":{\"_index\":\"medcl1\",\"_type\":\"doc1\",\"_id\":\"newId\"}}")
}

func TestGetUrlLevelMeta(t *testing.T) {

	pathStr:="/index/_bulk"
	pathArray := strings.FieldsFunc(pathStr, func(c rune) bool {
						return c=='/'
					} )
	fmt.Println(pathArray,len(pathArray))

	pathArray=strings.Split(pathStr,"/")
	fmt.Println(pathArray,len(pathArray))

	tindex, ttype := elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"")

	pathStr = "/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"")
	assert.Equal(t,ttype,"")

	pathStr = "//_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"")
	assert.Equal(t,ttype,"")

	pathStr = "/index/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"")

	pathStr = "//index/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"")

	pathStr = "//index//_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"")

	pathStr = "/index/doc/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"doc")

	pathStr = "//index/doc/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex,ttype)
	assert.Equal(t,tindex,"index")
	assert.Equal(t,ttype,"doc")
}
