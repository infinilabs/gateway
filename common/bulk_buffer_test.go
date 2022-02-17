package common

import (
	"fmt"
	"testing"
)

func TestBulkBuffer_Add(t *testing.T) {
	buffer:= AcquireBulkBuffer()
	buffer.Add("0,1",[]byte("message 0,1"))
	buffer.Add("0,2",[]byte("message 0,2"))
	buffer.Add("0,3",[]byte("message 0,3"))
	fmt.Println(buffer.MessageIDs[0])
	fmt.Println(buffer.Buffer.String())
}
