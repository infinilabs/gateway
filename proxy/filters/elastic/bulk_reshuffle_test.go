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
	"github.com/magiconair/properties/assert"
	"infini.sh/framework/core/elastic"
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
	data := []byte("{\"delete\":{\"_index\":\"idx-familycloud-stdfile2\",\"_id\":\"1411aX3240ge17520221106010809oh0\",\"routing\":\"ab1daa0979a64f32994a81c0091b1577\"}}\n{ \"create\" : { \"_index\" : \"my_index\", \"_id\" : \"2\"} }\n{ \"field\" : \"value2\", \"home_location\": \"41.12,-71.34\"}")
	fmt.Println(string(data))
}

func TestParseBulkRequestWithOnlyDelete(t *testing.T) {
	data := []byte("{\"delete\":{\"_index\":\"idx-familycloud-stdfile2\",\"_id\":\"1411aX3240ge17520221106010809oh0\",\"routing\":\"ab1daa0979a64f32994a81c0091b1577\"}}\n")
	fmt.Println(string(data))
}

// 有 partition 和没有 partition 可能有不同的解析行为
func TestBulkReshuffle_MixedRequests(t *testing.T) {
	data := "{\"update\":{\"_index\":\"idx-50\",\"_id\":\"ceq16t3q50k2vhtav6f0\",\"routing\":\"1513594400\",\"retry_on_conflict\":3}}\n{\"doc\":{\"address\":\"\"}}\n{\"delete\":{\"_index\":\"idx-50\",\"_id\":\"ceq16t3q50k2vhtav6g0\",\"routing\":\"1513594401\"}}\n{ \"create\" : { \"_index\" : \"idx-50\", \"_id\" : \"ceq16t3q50k2vhtav6gg\"} }\n{ \"field\" : \"value2\", \"home_location\": \"41.12,-71.34\"}\n"
	fmt.Println(string(data))
}

func TestGetUrlLevelMeta(t *testing.T) {

	pathStr := "/index/_bulk"
	pathArray := strings.FieldsFunc(pathStr, func(c rune) bool {
		return c == '/'
	})
	fmt.Println(pathArray, len(pathArray))

	pathArray = strings.Split(pathStr, "/")
	fmt.Println(pathArray, len(pathArray))

	tindex, ttype := elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "")

	pathStr = "/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "")
	assert.Equal(t, ttype, "")

	pathStr = "//_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "")
	assert.Equal(t, ttype, "")

	pathStr = "/index/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "")

	pathStr = "//index/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "")

	pathStr = "//index//_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "")

	pathStr = "/index/doc/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "doc")

	pathStr = "//index/doc/_bulk"
	tindex, ttype = elastic.ParseUrlLevelBulkMeta(pathStr)
	fmt.Println(tindex, ttype)
	assert.Equal(t, tindex, "index")
	assert.Equal(t, ttype, "doc")
}
