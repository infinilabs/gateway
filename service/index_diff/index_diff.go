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
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"runtime"
	log "src/github.com/cihub/seelog"
	"src/github.com/segmentio/fasthash/fnv1a"
	"strings"
	"sync"
)

type CompareItem struct {
	Doc      interface{}
	DiffType string
	Key      string
	Hash     string
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

type CompareChan struct {
	msgAChan chan CompareItem
	msgBChan chan CompareItem
	stopChan chan struct{}
}

func (t *CompareChan) addMsgA(msg CompareItem) bool {
	select {
	case <-t.stopChan:
		return false
	default:
	}
	select {
	case <-t.stopChan:
		return false
	case t.msgAChan <- msg:
	}
	return true
}

func (t *CompareChan) addMsgB(msg CompareItem) bool {
	select {
	case <-t.stopChan:
		return false
	default:
	}

	select {
	case <-t.stopChan:
		return false
	case t.msgBChan <- msg:
	}
	return true
}

func (t *CompareChan) processMsg(diffQueue string) {
	var msgA, msgB CompareItem

MOVEALL:
	msgA = <-t.msgAChan
	msgB = <-t.msgBChan

	if global.Env().IsDebug{
		log.Trace(msgA," vs ",msgB)
	}

	onlyInA := []*CompareItem{}
	onlyInB := []*CompareItem{}
	diffInBoth := []*CompareItem{}

COMPARE:
	result := msgA.CompareKey(&msgB)
	if result > 0 {
		onlyInB = append(onlyInB, &msgB)
		msgB.DiffType = "OnlyInTarget"
		queue.Push(diffQueue, util.MustToJSONBytes(msgB))
		if global.Env().IsDebug {
			fmt.Println(" :", msgB)
		}
		msgB = <-t.msgBChan
		goto COMPARE
	} else if result < 0 { // 1 < 2
		msgA.DiffType = "OnlyInSource"
		queue.Push(diffQueue, util.MustToJSONBytes(msgA))
		onlyInA = append(onlyInA, &msgA)
		if global.Env().IsDebug {
			fmt.Println(msgA, ": ")
		}
		msgA = <-t.msgAChan
		goto COMPARE
	} else {
		if msgA.CompareHash(&msgB) != 0 {
			if global.Env().IsDebug {
				fmt.Println(msgA, "!=", msgB)
			}
			msgB.DiffType = "DiffContent"
			queue.Push(diffQueue, util.MustToJSONBytes(msgA))
			diffInBoth = append(diffInBoth, &msgA)
		} else {
			if global.Env().IsDebug {
				log.Trace(msgA,"==",msgB)
			}
		}
		goto MOVEALL
	}

	//TODO timeout for last elements check
}

type IndexDiffModule struct {
}

func (this IndexDiffModule) Name() string {
	return "index_diff"
}

type Config struct {
	Enabled    bool   `config:"enabled"`
	BufferSize int    `config:"buffer_size"`
	DiffQueue  string `config:"diff_queue"`
	Source     struct {
		Elasticsearch string `config:"elasticsearch"`
		InputQueue    string `config:"input_queue"`
	} `config:"source"`

	Target struct {
		InputQueue string `config:"input_queue"`
	} `config:"target"`
}

var diffConfig = Config{
	BufferSize: 100,
	DiffQueue:  "diff_result",
}

var wg sync.WaitGroup
var testChan CompareChan

func (module IndexDiffModule) Setup(cfg *config.Config) {

	ok, err := env.ParseConfig("index_diff", &diffConfig)
	if ok && err != nil {
		panic(err)
	}

	testChan = CompareChan{
		msgAChan: make(chan CompareItem, diffConfig.BufferSize),
		msgBChan: make(chan CompareItem, diffConfig.BufferSize),
		stopChan: make(chan struct{}),
	}
}

func (module IndexDiffModule) Start() error {

	if !diffConfig.Enabled {
		return nil
	}

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

		go func() {
			for v := range queue.ReadChan(diffConfig.Source.InputQueue) {
				doc := map[string]interface{}{}
				util.FromJSONBytes(v, &doc)
				h1 := fnv1a.HashBytes64(util.MustToJSONBytes(doc["_source"]))
				hash := util.MustToJSONBytes(h1)
				delete(doc, "_score")
				delete(doc, "_source")
				delete(doc, "sort")
				item := CompareItem{
					Doc:  doc,
					Key:  fmt.Sprintf("%v", doc["_id"]),
					Hash: fmt.Sprintf("%v", string(hash)),
				}
				testChan.addMsgA(item)
			}
		}()

		go func() {
			for v := range queue.ReadChan(diffConfig.Target.InputQueue) {
				doc := map[string]interface{}{}
				util.FromJSONBytes(v, &doc)
				h1 := fnv1a.HashBytes64(util.MustToJSONBytes(doc["_source"]))
				hash := util.MustToJSONBytes(h1)
				delete(doc, "_score")
				delete(doc, "_source")
				delete(doc, "sort")
				item := CompareItem{
					Doc:  doc,
					Key:  fmt.Sprintf("%v", doc["_id"]),
					Hash: fmt.Sprintf("%v", string(hash)),
				}
				testChan.addMsgB(item)
			}
		}()

		wg.Add(1)
		go testChan.processMsg(diffConfig.DiffQueue)
		wg.Wait()

	}()

	return nil
}

func (module IndexDiffModule) Stop() error {
	if !diffConfig.Enabled {
		return nil
	}
	close(testChan.stopChan)
	wg.Done()
	return nil
}
