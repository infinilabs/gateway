/*
Copyright Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package index_diff

import (
	"fmt"
	"github.com/bsm/extsort"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type CompareItem struct {
	Doc  interface{} `json:"doc,omitempty"`
	Key  string      `json:"key,omitempty"`
	Hash string      `json:"hash,omitempty"`
}

type DiffResult struct {
	DiffType string       `json:"type,omitempty"`
	Source   *CompareItem `json:"source,omitempty"`
	Target   *CompareItem `json:"target,omitempty"`
}

func (a *CompareItem) CompareKey(b *CompareItem) int {
	v := strings.Compare(a.Key, b.Key)
	return v
}

func (a *CompareItem) CompareHash(b *CompareItem) int {
	return strings.Compare(a.Hash, b.Hash)
}

func NewCompareItem(key, hash string) CompareItem {
	item := CompareItem{Key: key, Hash: hash}
	return item
}

func processMsg(diffQueue string) {
	var msgA, msgB CompareItem

	//distance:=0

MOVEALL:
	b1, err := queue.Pop(diffConfig.SortedLeftQueue)
	if err != nil {
		panic(err)
	}
	util.MustFromJSONBytes(b1, &msgA)

	b2, err := queue.Pop(diffConfig.SortedRightQueue)
	if err != nil {
		panic(err)
	}
	util.MustFromJSONBytes(b2, &msgB)

COMPARE:
	result := msgA.CompareKey(&msgB)

	if global.Env().IsDebug {
		log.Trace(result, " - ", msgA, " vs ", msgB)
	}
	//distance++
	//if msgA.Key=="c46krqcgq9s2jd9v9tig"||msgB.Key=="c46krqcgq9s2jd9v9tig"{
	//	distance=0
	//}

	if result > 0 {

		result := DiffResult{Target: &msgB}
		result.DiffType = "OnlyInTarget"
		queue.Push(diffQueue, util.MustToJSONBytes(result))
		if global.Env().IsDebug {
			log.Trace("OnlyInTarget :", msgB)
		}
		b2, err := queue.Pop(diffConfig.SortedRightQueue)
		if err != nil {
			panic(err)
		}
		util.MustFromJSONBytes(b2, &msgB)
		goto COMPARE
	} else if result < 0 { // 1 < 2

		result := DiffResult{Source: &msgA}
		result.DiffType = "OnlyInSource"
		queue.Push(diffQueue, util.MustToJSONBytes(result))
		if global.Env().IsDebug {
			log.Trace(msgA, ": OnlyInSource")
		}
		b1, err := queue.Pop(diffConfig.SortedLeftQueue)
		if err != nil {
			panic(err)
		}
		util.MustFromJSONBytes(b1, &msgA)
		goto COMPARE
	} else {
		//doc exists, compare hash
		if msgA.CompareHash(&msgB) != 0 {
			//fmt.Println(msgA, "!=", msgB)
			if global.Env().IsDebug {
				log.Trace(msgA, "!=", msgB)
			}
			result := DiffResult{Target: &msgB, Source: &msgA}
			result.DiffType = "DiffBoth"
			queue.Push(diffQueue, util.MustToJSONBytes(result))
		} else {
			if global.Env().IsDebug {
				log.Trace(msgA, "==", msgB)
			}
		}
		goto MOVEALL
	}
}

type IndexDiffModule struct {
}

func (this IndexDiffModule) Name() string {
	return "index_diff"
}

type Config struct {
	Enabled            bool   `config:"enabled"`
	TextReportEnabled  bool   `config:"text_report"`
	KeepSourceInResult bool   `config:"keep_source"`
	BufferSize         int    `config:"buffer_size"`
	DiffQueue          string `config:"diff_queue"`
	SortedLeftQueue    string `config:"sorted_source"`
	SortedRightQueue   string `config:"sorted_target"`
	Source             struct {
		InputQueue    string `config:"input_queue"`
	} `config:"source"`

	Target struct {
		InputQueue string `config:"input_queue"`
	} `config:"target"`
}

var diffConfig = Config{
	TextReportEnabled:true,
	BufferSize:       1,
	DiffQueue:        "diff_result",
	SortedLeftQueue:  "sorted_source",
	SortedRightQueue: "sorted_target",
}

var wg sync.WaitGroup

func (module IndexDiffModule) Setup(cfg *config.Config) {

	ok, err := env.ParseConfig("index_diff", &diffConfig)
	if ok && err != nil {
		panic(err)
	}
}

func (module IndexDiffModule) Start() error {

	if !diffConfig.Enabled {
		return nil
	}

	//opt := nutsdb.DefaultOptions
	//opt.Dir = path.Join(global.Env().GetDataDir(), "index_diff")
	//var err error
	//db, err := nutsdb.Open(opt)
	//if err != nil {
	//	panic(err)
	//}

	go func() {
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error in index_diff service", v)
				}
			}
		}()

		//build sorted file
		//go func() {

			files:=[]string{"source_docs","target_docs"}
			for _,v:=range files{
				sorter := extsort.New(nil)
				defer sorter.Close()
				file:=path.Join(global.Env().GetDataDir(),"diff",v)
				file1:=path.Join(global.Env().GetDataDir(),"diff",v+"_sorted")
				lines:=util.FileGetLines(file)
				for _,v:=range lines{
					_ = sorter.Append([]byte(v))
				}

				// Sort and iterate.
				iter, err := sorter.Sort()
				if err != nil {
					panic(err)
				}
				defer iter.Close()

				for iter.Next() {
					//fmt.Println(string(iter.Data()))
					util.FileAppendNewLine(file1,string(iter.Data()))
				}
				if err := iter.Err(); err != nil {
					panic(err)
				}
			}

		//}()

		//popup source sorted list
		//go func() {

			file1:=path.Join(global.Env().GetDataDir(),"diff","source_docs_sorted")
			lines:=util.FileGetLines(file1)
			for _,v:=range lines{
				arr:=strings.Split(v,",")
				id:=arr[0]
				hash:=arr[1]
				item := CompareItem{
					//Doc:  doc,
					Key:  fmt.Sprintf("%v", id),
					Hash: fmt.Sprintf("%v", (hash)),
				}
				queue.Push(diffConfig.SortedLeftQueue, util.MustToJSONBytes(item))
			}

		//}()

		//popup target sorted list
		//go func() {

			file1=path.Join(global.Env().GetDataDir(),"diff","target_docs_sorted")
			lines=util.FileGetLines(file1)
			for _,v:=range lines{
				arr:=strings.Split(v,",")
				id:=arr[0]
				hash:=arr[1]
				item := CompareItem{
					//Doc:  doc,
					Key:  fmt.Sprintf("%v", id),
					Hash: fmt.Sprintf("%v", (hash)),
				}
				queue.Push(diffConfig.SortedRightQueue, util.MustToJSONBytes(item))
			}
		//}()

		if diffConfig.TextReportEnabled {
			go func() {
				path1 := path.Join(global.Env().GetLogDir(), "diff_result", fmt.Sprintf("%v_vs_%v", diffConfig.Source.InputQueue, diffConfig.Target.InputQueue))
				os.MkdirAll(path1, 0775)
				file := path.Join(path1, util.FormatTimeForFileName(time.Now())+".log")
				str := "    ___ _  __  __     __                 _ _   \n"
				str += "   /   (_)/ _|/ _|   /__\\ ___  ___ _   _| | |_ \n"
				str += "  / /\\ / | |_| |_   / \\/// _ \\/ __| | | | | __|\n"
				str += " / /_//| |  _|  _| / _  \\  __/\\__ \\ |_| | | |_ \n"
				str += "/___,' |_|_| |_|   \\/ \\_/\\___||___/\\__,_|_|\\__|\n"

				strBuilder := strings.Builder{}
				leftBuilder := strings.Builder{}
				rightBuilder := strings.Builder{}
				bothBuilder := strings.Builder{}
				strBuilder.WriteString(str)

				var i,left,right,both int

			WAIT:
				timeOut := 1 * time.Second
				for {
					//if queue.Depth(diffConfig.Source.InputQueue) > 0 ||
					//	queue.Depth(diffConfig.SortedLeftQueue) > 0 ||
					//	queue.Depth(diffConfig.SortedRightQueue) > 0 ||
					//	queue.Depth(diffConfig.Target.InputQueue) > 0 {
					//	time.Sleep(10 * time.Second)
					//	goto WAIT
					//}

					v, timeout, err := queue.PopTimeout(diffConfig.DiffQueue, timeOut)
					if timeout {
						if queue.Depth(diffConfig.Source.InputQueue) > 0 ||
							queue.Depth(diffConfig.SortedLeftQueue) > 0 ||
							queue.Depth(diffConfig.SortedRightQueue) > 0 ||
							queue.Depth(diffConfig.Target.InputQueue) > 0 {
							time.Sleep(10 * time.Second)
							goto WAIT
						}
						goto RESULT
					}

					i++
					doc := DiffResult{}
					err = util.FromJSONBytes(v, &doc)
					if err != nil {
						log.Error(err)
						return
					}
					docID := ""
					docHash := ""
					if doc.Source != nil {
						docID = doc.Source.Key
						docHash = doc.Source.Hash
					}
					if doc.Target != nil {
						docID = doc.Target.Key
						docHash = doc.Target.Hash
					}

					switch doc.DiffType {
					case "OnlyInSource":
						left++
						leftBuilder.WriteString(fmt.Sprintf("doc:%v, hash:%v\n", docID, docHash))
						break
					case "OnlyInTarget":
						right++
						rightBuilder.WriteString(fmt.Sprintf("doc:%v, hash:%v\n", docID, docHash))
						break
					case "DiffBoth":
						both++
						bothBuilder.WriteString(fmt.Sprintf("doc:%v, hash:%v vs %v\n", docID, doc.Source.Hash, doc.Target.Hash))
						break
					}
				}
			RESULT:
				fmt.Println(strBuilder.String())
				util.FileAppendNewLine(file, strBuilder.String())

				if leftBuilder.Len() > 0 {
					str:=fmt.Sprintf("%v Documents diff in left side:",left)
					fmt.Println(str)
					fmt.Println(leftBuilder.String())

					util.FileAppendNewLine(file, str)
					util.FileAppendNewLine(file, leftBuilder.String())
				}
				if rightBuilder.Len() > 0 {

					str:=fmt.Sprintf("%v Documents diff in right side:",right)
					fmt.Println(str)
					fmt.Println(rightBuilder.String())

					util.FileAppendNewLine(file, str)
					util.FileAppendNewLine(file, rightBuilder.String())
				}
				if bothBuilder.Len() > 0 {

					str:=fmt.Sprintf("%v Documents diff in both side:",both)
					fmt.Println(str)
					fmt.Println(bothBuilder.String())

					util.FileAppendNewLine(file, str)
					util.FileAppendNewLine(file, bothBuilder.String())
				}

			}()
		}



		wg.Add(1)

		go processMsg(diffConfig.DiffQueue)
		wg.Wait()

	}()

	return nil
}

func (module IndexDiffModule) Stop() error {
	if !diffConfig.Enabled {
		return nil
	}
	//close(testChan.stopChan)
	wg.Done()
	return nil
}
