package scroll

import (
	"fmt"
	"infini.sh/framework/lib/bytebufferpool"
	"testing"
)

func TestHash(t *testing.T) {

	buffer:=bytebufferpool.Get()
	data:="Just create a slice of 128 length and find maximum frequency at every iteration.\n"
	data1:="Just create a slice of 128 length and find maximum frequency at every iteration1.\n"
	hash:=frequencySort(buffer,data)

	hash1:=frequencySort(buffer,data1)
	fmt.Println(hash)
	fmt.Println(hash1)
}