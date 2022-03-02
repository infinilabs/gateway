package index_diff
//
//import (
//	"fmt"
//	"github.com/stretchr/testify/assert"
//	"sync"
//	"testing"
//	"time"
//)
//
//func TestCompareItems(t *testing.T) {
//
//	a:=[]CompareItem{
//		NewCompareItem("1", "1"), //diff left
//		NewCompareItem("2", "1"),
//		NewCompareItem("3", "1"), //diff left
//		NewCompareItem("4", "1"),
//		NewCompareItem("5", "1"),
//		NewCompareItem("9", "1"),
//		NewCompareItem("11", "1"), //diff left
//		NewCompareItem("12", "1"), //diff both
//	}
//
//	b:=[]CompareItem{
//		NewCompareItem("2","1"),
//		NewCompareItem("4","1"),
//		NewCompareItem("5","1"),
//		NewCompareItem("8","1"), //diff right
//		NewCompareItem("9","1"),
//		NewCompareItem("10","1"), //diff right
//		NewCompareItem("12","2"),}
//	module:= IndexDiffProcessor{
//		config: Config{
//			SourceInputQueue: "source",
//			TargetInputQueue: "target",
//		},
//		testChan : CompareTask{
//		msgChans: map[string]chan CompareItem{},
//		stopChan: make(chan struct{}),
//	},
//	}
//
//	module.testChan.msgChans[module.config.GetSortedLeftQueue()]=make(chan CompareItem)
//	module.testChan.msgChans[module.config.GetSortedRightQueue()]=make(chan CompareItem)
//
//	a1:=[]string{}
//	m:=map[string]string{}
//	go module.processMsg(func(result DiffResult) {
//		key:=result.DiffType+","+result.Key
//		fmt.Println(result.DiffType,",",result.Key)
//		a1=append(a1,key)
//		m[key]=result.Key
//	})
//
//	wg:=sync.WaitGroup{}
//	wg.Add(1)
//	go func() {
//		for _,v:=range a{
//			//fmt.Println("InputA:",v.Key)
//			module.testChan.msgChans[module.config.GetSortedLeftQueue()]<- v
//		}
//		wg.Done()
//	}()
//
//	wg.Add(1)
//	go func() {
//		for _,v:=range b{
//			//fmt.Println("InputB:",v.Key)
//			module.testChan.msgChans[module.config.GetSortedRightQueue()]<- v
//		}
//		wg.Done()
//	}()
//
//	wg.Wait()
//	time.Sleep(1*time.Second)
//
//	assert.Equal(t, 6,len(a1))
//
//	assert.Equal(t,"1",m["OnlyInSource,1"])
//	assert.Equal(t,"3",m["OnlyInSource,3"])
//	assert.Equal(t,"11",m["OnlyInSource,11"])
//	assert.Equal(t,"8",m["OnlyInTarget,8"])
//	assert.Equal(t,"10",m["OnlyInTarget,10"])
//	assert.Equal(t,"12",m["DiffBoth,12"])
//
//	//OnlyInSource , 1
//	//OnlyInSource , 3
//	//OnlyInTarget , 8
//	//OnlyInTarget , 10
//	//OnlyInSource , 11
//	//DiffBoth , 12
//
//}
