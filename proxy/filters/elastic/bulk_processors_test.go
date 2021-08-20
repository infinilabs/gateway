package elastic

import (
	"fmt"
	"testing"
)

func TestBulkWalkLines(t *testing.T) {
	bulkRequests:= "{ \"index\" : { \"_index\" : \"medcl-test\",\"_type\" : \"doc\", \"_id\" : \"id1\" } }\n{ \"id\" : \"123\",\"field1\" : \"user2\",\"ip\" : \"123\" }\n"
	bulkRequests+= "{ \"index\" : { \"_index\" : \"medcl-test\",\"_type\" : \"doc\", \"_id\" : \"id2\" } }\n{ \"id\" : \"345\",\"field1\" : \"user1\",\"ip\" : \"456\" }\n"
	bulkRequests+= "{ \"index\" : { \"_index\" : \"test\", \"_id\" : \"1\" } }\n { \"field1\" : \"value1\" }\n" +
		"{ \"delete\" : { \"_index\" : \"test\", \"_id\" : \"2\" } }\n" +
		"{ \"create\" : { \"_index\" : \"test\", \"_id\" : \"3\" } }\n{ \"field1\" : \"value3\" }\n" +
		"{ \"update\" : {\"_id\" : \"1\", \"_index\" : \"test\"} }\n{ \"doc\" : {\"field2\" : \"value2\"} }\n"


	WalkBulkRequests([]byte(bulkRequests), func(eachLine []byte) (skipNextLine bool) {
		//fmt.Println(string(eachLine))
		return false
	}, func(metaBytes []byte,actionStr,index,typeName,id string) (err error) {
		fmt.Println(string(metaBytes))
		return nil
	}, func(payloadBytes []byte) {
		fmt.Println(string(payloadBytes))
	})
}
