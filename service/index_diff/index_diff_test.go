package index_diff

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCompareItems(t *testing.T) {

	a:=[]CompareItem{
		NewCompareItem("1", "1"),//diff left
		NewCompareItem("2", "1"),
		NewCompareItem("3", "1"),//diff left
		NewCompareItem("4", "1"),
		NewCompareItem("5", "1"),
		NewCompareItem("9", "1"),
		NewCompareItem("11", "1"),//diff left
		NewCompareItem("12", "1"),//diff both
	}

	b:=[]CompareItem{
			NewCompareItem("2","1"),
			NewCompareItem("4","1"),
			NewCompareItem("5","1"),
			NewCompareItem("8","1"),//diff right
			NewCompareItem("9","1"),
			NewCompareItem("10","1"),//diff right
			NewCompareItem("12","2"),}

	testChan = CompareChan{
		msgChans: map[string]chan CompareItem{},
		stopChan: make(chan struct{}),
	}

	testChan.msgChans[diffConfig.GetSortedLeftQueue()]=make(chan CompareItem)
	testChan.msgChans[diffConfig.GetSortedRightQueue()]=make(chan CompareItem)

	go processMsg(func(result DiffResult) {
		fmt.Println(result.DiffType,",",result.Key)
	})

	wg:=sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for _,v:=range a{
			//fmt.Println("InputA:",v.Key)
			testChan.msgChans[diffConfig.GetSortedLeftQueue()]<- v
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for _,v:=range b{
			//fmt.Println("InputB:",v.Key)
			testChan.msgChans[diffConfig.GetSortedRightQueue()]<- v
		}
		wg.Done()
	}()

	wg.Wait()
	time.Sleep(1*time.Second)

	//OnlyInSource , 1
	//OnlyInSource , 3
	//OnlyInTarget , 8
	//OnlyInTarget , 10
	//OnlyInSource , 11
	//DiffBoth , 12

}
