package floating_ip

import (
	"fmt"
	"testing"
)

func TestPingActiveNode(t *testing.T) {
	ok:=pingActiveNode("192.168.3.98")
	fmt.Println(ok)
}