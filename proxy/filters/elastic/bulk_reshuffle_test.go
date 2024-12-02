// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

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

func TestParseBulkRequestWithDelete(t *testing.T) {
	data:=[]byte("{\"delete\":{\"_index\":\"idx-familycloud-stdfile2\",\"_id\":\"1411aX3240ge17520221106010809oh0\",\"routing\":\"ab1daa0979a64f32994a81c0091b1577\"}}\n{ \"create\" : { \"_index\" : \"my_index\", \"_id\" : \"2\"} }\n{ \"field\" : \"value2\", \"home_location\": \"41.12,-71.34\"}")
	fmt.Println(string(data))
}

func TestParseBulkRequestWithOnlyDelete(t *testing.T) {
	data:=[]byte("{\"delete\":{\"_index\":\"idx-familycloud-stdfile2\",\"_id\":\"1411aX3240ge17520221106010809oh0\",\"routing\":\"ab1daa0979a64f32994a81c0091b1577\"}}\n")
	fmt.Println(string(data))
}

//有 partition 和没有 partition 可能有不同的解析行为
func TestBulkReshuffle_MixedRequests(t *testing.T) {
	data:="{\"update\":{\"_index\":\"idx-50\",\"_id\":\"ceq16t3q50k2vhtav6f0\",\"routing\":\"1513594400\",\"retry_on_conflict\":3}}\n{\"doc\":{\"address\":\"\"}}\n{\"delete\":{\"_index\":\"idx-50\",\"_id\":\"ceq16t3q50k2vhtav6g0\",\"routing\":\"1513594401\"}}\n{ \"create\" : { \"_index\" : \"idx-50\", \"_id\" : \"ceq16t3q50k2vhtav6gg\"} }\n{ \"field\" : \"value2\", \"home_location\": \"41.12,-71.34\"}\n"
	fmt.Println(string(data))
}

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

	action, indexb, typeb, idb,_ ,_ := elastic.ParseActionMeta(data)
	fmt.Println(string(action), string(indexb), string(idb))
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")


	data = []byte("{\"index\":{\"_type\":\"_doc\",\"_id\":\"GZq-bnYBC53QmW9Kk2ve\",\"_index\":\"medcl1\"}}")

	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")


	data = []byte("{\"index\":{\"_id\":\"GZq-bnYBC53QmW9Kk2ve\",\"_type\":\"_doc\",\"_index\":\"medcl1\"}}")

	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"medcl1")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"GZq-bnYBC53QmW9Kk2ve")

	data = []byte("{\"index\":{\"_index\":\"test\",\"_type\":\"doc\"}}")
	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"index")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{\"delete\":{\"_index\":\"test\",\"_type\":\"_doc\"}}")
	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"delete")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{\"create\":{\"_index\":\"test\",\"_type\":\"_doc\"}}")
	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"create")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"_doc")
	assert.Equal(t,string(idb),"")

	data = []byte("{ \"update\" : {\"_id\" : \"1\", \"_index\" : \"test\"} }")
	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

	fmt.Println(string(action), string(indexb), string(idb), )
	assert.Equal(t,string(action),"update")
	assert.Equal(t,string(indexb),"test")
	assert.Equal(t,string(typeb),"")
	assert.Equal(t,string(idb),"1")

	data = []byte("{ \"update\" : {\"_index\" : \"test\"} }")
	action, indexb, typeb, idb,_,_  = elastic.ParseActionMeta(data)

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


