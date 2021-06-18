package index_diff

import (
	"testing"
	"time"
)

func TestCompareItems(t *testing.T) {

	a:=[]CompareItem{
		NewCompareItem("1", "1"),
		NewCompareItem("2", "1"),
		NewCompareItem("3", "1"),
		NewCompareItem("4", "1"),
		NewCompareItem("5", "1"),
		NewCompareItem("9", "1"),
		NewCompareItem("11", "1"),
		NewCompareItem("12", "1"),
	}

	b:=[]CompareItem{
			NewCompareItem("2","1"),
			NewCompareItem("4","1"),
			NewCompareItem("5","1"),
			NewCompareItem("8","1"),
			NewCompareItem("9","1"),
			NewCompareItem("10","1"),
			NewCompareItem("12","2"),}

	buffer:=10
	testChan:= CompareChan{
		msgAChan: make(chan CompareItem,buffer),
		msgBChan: make(chan CompareItem,buffer),
		stopChan: make(chan struct{}),
	}

	go testChan.processMsg()

	go func() {
		for _,v:=range a{
			testChan.addMsgA(v)
		}
	}()

	go func() {
		for _,v:=range b{
			testChan.addMsgB(v)
		}
	}()

	time.Sleep(time.Minute)

}
