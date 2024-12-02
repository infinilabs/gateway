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

package index_diff

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bsm/extsort"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/bytebufferpool"
)

type CompareItem struct {
	Doc  interface{} `json:"doc,omitempty"`
	Key  string      `json:"key,omitempty"`
	Hash string      `json:"hash,omitempty"`
}

type DiffResult struct {
	DiffType string       `json:"type,omitempty"`
	Key      string       `json:"key,omitempty"`
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

func init() {
	pipeline.RegisterProcessorPlugin("index_diff", New)
}

func NewCompareItem(key, hash string) CompareItem {
	item := CompareItem{Key: key, Hash: hash}
	return item
}

func handleDiffResult(diffType string, msgA, msgB *CompareItem, diffResultHandler func(DiffResult)) {
	result := DiffResult{Target: msgB, Source: msgA}
	result.DiffType = diffType
	if msgA != nil {
		result.Key = msgA.Key
	} else if msgB != nil {
		result.Key = msgB.Key
	}
	diffResultHandler(result)
}

func (processor *IndexDiffProcessor) processMsg(partitionID int, diffResultHandler func(DiffResult)) {
	var msgA, msgB CompareItem
	var okA, okB bool
	var readMode = 0 // 0-all 1-A 2-B
	var (
		sourceChan = processor.testChans[partitionID].msgChans[processor.config.GetSortedLeftQueue(partitionID)]
		targetChan = processor.testChans[partitionID].msgChans[processor.config.GetSortedRightQueue(partitionID)]
	)

	for {
		if readMode == 0 || readMode == 1 {
			msgA, okA = <-sourceChan
		}
		if readMode == 0 || readMode == 2 {
			msgB, okB = <-targetChan
		}

		// fmt.Println("Pop A:", msgA.Key)
		// fmt.Println("Pop B:", msgB.Key)
		if !okA && !okB {
			return
		}
		if !okA {
			handleDiffResult("OnlyInTarget", nil, &msgB, diffResultHandler)
			for msgB = range targetChan {
				handleDiffResult("OnlyInTarget", nil, &msgB, diffResultHandler)
			}
			return
		}
		if !okB {
			handleDiffResult("OnlyInSource", &msgA, nil, diffResultHandler)
			for msgA = range sourceChan {
				handleDiffResult("OnlyInSource", &msgA, nil, diffResultHandler)
			}
			return
		}

		result := msgA.CompareKey(&msgB)

		//fmt.Println("A:",msgA.Key," vs B:",msgB.Key,"=",result)
		if global.Env().IsDebug {
			log.Trace(result, " - ", msgA, " vs ", msgB)
		}

		if result > 0 {
			handleDiffResult("OnlyInTarget", nil, &msgB, diffResultHandler)
			if global.Env().IsDebug {
				log.Trace("OnlyInTarget :", msgB)
			}

			readMode = 2
		} else if result < 0 { // 1 < 2
			handleDiffResult("OnlyInSource", &msgA, nil, diffResultHandler)
			if global.Env().IsDebug {
				log.Trace(msgA, ": OnlyInSource")
			}

			readMode = 1
		} else {
			//doc exists, compare hash
			if msgA.CompareHash(&msgB) != 0 {
				if global.Env().IsDebug {
					log.Trace(msgA, "!=", msgB)
				}
				handleDiffResult("DiffBoth", &msgA, &msgB, diffResultHandler)

			} else {
				if global.Env().IsDebug {
					log.Trace(msgA, "==", msgB)
				}
				//handleDiffResult("Equal", &msgA, &msgB, diffResultHandler)
			}
			readMode = 0
		}
	}
}

type DiffStat struct {
	Count int
	Keys  []string
}

type IndexDiffProcessor struct {
	config    Config
	testChans []CompareChan
	wg        sync.WaitGroup

	onlyInSource DiffStat
	onlyInTarget DiffStat
	diffBoth     DiffStat

	statLock sync.Mutex
}

func New(c *config.Config) (pipeline.Processor, error) {
	diffConfig := Config{
		TextReportEnabled: true,
		PartitionSize:     10,
		BufferSize:        1,
		SourceInputQueue:  "source",
		TargetInputQueue:  "target",
		DiffQueue:         "diff_result",
	}

	if err := c.Unpack(&diffConfig); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of index_diff processor: %s", err)
	}

	if diffConfig.Queue != nil {
		diffConfig.DiffQueue = diffConfig.Queue.Name
	}

	diffs := []CompareChan{}
	for i := 0; i < diffConfig.PartitionSize; i++ {
		diff := CompareChan{}
		diff.msgChans = map[string]chan CompareItem{}
		diff.stopChan = make(chan struct{})

		diff.msgChans[diffConfig.GetSortedLeftQueue(i)] = make(chan CompareItem, diffConfig.BufferSize)
		diff.msgChans[diffConfig.GetSortedRightQueue(i)] = make(chan CompareItem, diffConfig.BufferSize)

		diffs = append(diffs, diff)
	}

	diff := &IndexDiffProcessor{
		config:    diffConfig,
		testChans: diffs,
	}

	return diff, nil

}

type CompareChan struct {
	msgChans map[string]chan CompareItem
	stopChan chan struct{}
}

func (processor *IndexDiffProcessor) Name() string {
	return "index_diff"
}

type OutputQueueConfig struct {
	Name   string                 `config:"name"`
	Labels map[string]interface{} `config:"labels"`
}

type Config struct {
	PartitionSize      int  `config:"partition_size"`
	TextReportEnabled  bool `config:"text_report"`
	KeepSourceInResult bool `config:"keep_source"`
	BufferSize         int  `config:"buffer_size"`

	Queue            *OutputQueueConfig `config:"output_queue"`
	CleanOldFiles    bool               `config:"clean_old_files"`
	SourceInputQueue string             `config:"source_queue"`
	TargetInputQueue string             `config:"target_queue"`
	// DEPRECATED, use `output_queue` instead
	DiffQueue string `config:"diff_queue"`
}

func (cfg *Config) GetLeftQueue(partitionID int) string {
	return cfg.SourceInputQueue + "-" + util.ToString(partitionID)
}

func (cfg *Config) GetRightQueue(partitionID int) string {
	return cfg.TargetInputQueue + "-" + util.ToString(partitionID)
}

func (cfg *Config) GetSortedLeftQueue(partitionID int) string {
	return cfg.GetLeftQueue(partitionID) + "_sorted"
}

func (cfg *Config) GetSortedRightQueue(partitionID int) string {
	return cfg.GetRightQueue(partitionID) + "_sorted"
}

func (processor *IndexDiffProcessor) Process(ctx *pipeline.Context) error {
	log.Infof("start index diff.")

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
				ctx.RecordError(fmt.Errorf("index diff panic: %v", r))
			}
		}
	}()

	for i := 0; i < processor.config.PartitionSize; i++ {
		processor.wg.Add(1)
		j := i
		task.RunWithinGroup("index_diff", func(ctx context.Context) error {
			defer processor.wg.Done()
			processor.sortFiles(processor.config.GetLeftQueue(j), j)
			return nil
		})
	}
	for i := 0; i < processor.config.PartitionSize; i++ {
		processor.wg.Add(1)
		j := i
		task.RunWithinGroup("index_diff", func(ctx context.Context) error {
			defer processor.wg.Done()
			processor.sortFiles(processor.config.GetRightQueue(j), j)
			return nil
		})
	}

	queueConfig := &queue.QueueConfig{}
	queueConfig.Source = "dynamic"
	queueConfig.Labels = util.MapStr{}
	queueConfig.Labels["type"] = "index_diff"
	if processor.config.Queue != nil {
		for k, v := range processor.config.Queue.Labels {
			queueConfig.Labels[k] = v
		}
	}
	queueConfig.Name = processor.config.DiffQueue
	queue.RegisterConfig(queueConfig)

	for i := 0; i < processor.config.PartitionSize; i++ {
		processor.wg.Add(1)
		j := i
		task.RunWithinGroup("index_diff", func(ctx context.Context) error {
			defer processor.wg.Done()

			processor.processMsg(j, func(result DiffResult) {
				processor.updateStats(&result)
				queue.Push(queue.GetOrInitConfig(processor.config.DiffQueue), util.MustToJSONBytes(result))
			})
			return nil
		})
	}

	processor.wg.Wait()

	if processor.config.TextReportEnabled {
		processor.wg.Add(1)
		task.RunWithinGroup("index_diff", func(ctx context.Context) error {
			defer processor.wg.Done()
			processor.generateTextReport()
			return nil
		})
		processor.wg.Wait()
	}

	log.Infof("index diff finished.")

	processor.statLock.Lock()
	defer processor.statLock.Unlock()

	ctx.PutValue("index_diff.only_in_target.count", processor.onlyInTarget.Count)
	ctx.PutValue("index_diff.only_in_target.keys", processor.onlyInTarget.Keys)
	ctx.PutValue("index_diff.only_in_source.count", processor.onlyInSource.Count)
	ctx.PutValue("index_diff.only_in_source.keys", processor.onlyInSource.Keys)
	ctx.PutValue("index_diff.diff_both.count", processor.diffBoth.Count)
	ctx.PutValue("index_diff.diff_both.keys", processor.diffBoth.Keys)

	return nil
}

func (processor *IndexDiffProcessor) sortFiles(inputFile string, partitionID int) {
	defer func() {
		close(processor.testChans[partitionID].msgChans[inputFile+"_sorted"])
	}()

	buffer := bytebufferpool.Get("index_diff")
	defer bytebufferpool.Put("index_diff", buffer)

	//build sorted file
	sorter := extsort.New(nil)
	file := path.Join(global.Env().GetDataDir(), "diff", inputFile)
	sortedFile := path.Join(global.Env().GetDataDir(), "diff", inputFile+"_sorted")

	if !util.FileExists(file) {
		log.Warnf("dump file [%s] not found, skip diffing", file)
		return
	}
	if util.FileExists(sortedFile) {
		if processor.config.CleanOldFiles {
			err := util.FileDelete(sortedFile)
			log.Infof("deleting old sorted file [%s], err: %v", sortedFile, err)
		} else {
			log.Warnf("sorted file [%s] exists, you may need to remove it first", sortedFile)
		}
	}

	if !util.FileExists(sortedFile) {
		err := util.FileLinesWalk(file, func(bytes []byte) {
			_ = sorter.Append(bytes)
		})
		if err != nil {
			log.Error(err)
			return
		}

		defer sorter.Close()
		// Sort and iterate.
		iter, err := sorter.Sort()
		if err != nil {
			log.Error(err)
			return
		}
		defer iter.Close()

		for iter.Next() {
			buffer.Write(iter.Data())
			buffer.WriteByte('\n')

			if buffer.Len() > 10*1024 {
				util.FileAppendContentWithByte(sortedFile, buffer.Bytes())
				buffer.Reset()
			}
		}

		if buffer.Len() > 0 {
			util.FileAppendContentWithByte(sortedFile, buffer.Bytes())
		}

		if err := iter.Err(); err != nil {
			log.Error(err)
			return
		}
	}

	//popup sorted list
	err := util.FileLinesWalk(sortedFile, func(bytes []byte) {
		arr := strings.FieldsFunc(string(bytes), func(r rune) bool {
			return r == ','
		})
		if len(arr) < 2 {
			//log.Error("invalid line:", util.UnsafeBytesToString(bytes))
			return
		}
		id := arr[0]
		hash := arr[1]
		item := CompareItem{
			Key:  id,
			Hash: hash,
		}
		if processor.config.KeepSourceInResult && len(arr) > 2 {
			doc := arr[3]
			item.Doc = doc
		}
		processor.testChans[partitionID].msgChans[inputFile+"_sorted"] <- item
	})
	if err != nil {
		log.Error(err)
		return
	}
}

func (processor *IndexDiffProcessor) generateTextReport() {
	path1 := path.Join(global.Env().GetLogDir(), "diff_result", fmt.Sprintf("%v_vs_%v", processor.config.SourceInputQueue, processor.config.TargetInputQueue))
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

	var i, left, right, both, equal int

	timeOut := 1 * time.Second
	for {

		v, timeout, err := queue.PopTimeout(queue.GetOrInitConfig(processor.config.DiffQueue), timeOut)
		if timeout {
			break
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
		case "Equal":
			equal++
		}
	}
	fmt.Println(strBuilder.String())
	util.FileAppendNewLine(file, strBuilder.String())

	if leftBuilder.Len() > 0 {
		str := fmt.Sprintf("%v documents only exists in left side:", left)
		fmt.Println(str)
		fmt.Println(leftBuilder.String())

		util.FileAppendNewLine(file, str)
		util.FileAppendNewLine(file, leftBuilder.String())
	}
	if rightBuilder.Len() > 0 {

		str := fmt.Sprintf("%v documents only exists in right side:", right)
		fmt.Println(str)
		fmt.Println(rightBuilder.String())

		util.FileAppendNewLine(file, str)
		util.FileAppendNewLine(file, rightBuilder.String())
	}
	if bothBuilder.Len() > 0 {

		str := fmt.Sprintf("%v documents exists but diff in both side:", both)
		fmt.Println(str)
		fmt.Println(bothBuilder.String())

		util.FileAppendNewLine(file, str)
		util.FileAppendNewLine(file, bothBuilder.String())
	}

	if leftBuilder.Len() == 0 && rightBuilder.Len() == 0 && bothBuilder.Len() == 0 {
		fmt.Println("Congratulations, the two clusters are consistent!\n")
	}

}

func (processor *IndexDiffProcessor) updateStats(diff *DiffResult) {
	processor.statLock.Lock()
	defer processor.statLock.Unlock()
	switch diff.DiffType {
	case "OnlyInSource":
		processor.onlyInSource.Count += 1
		processor.onlyInSource.Keys = appendStrArr(processor.onlyInSource.Keys, 10, diff.Key)
	case "OnlyInTarget":
		processor.onlyInTarget.Count += 1
		processor.onlyInTarget.Keys = appendStrArr(processor.onlyInTarget.Keys, 10, diff.Key)
	case "DiffBoth":
		processor.diffBoth.Count += 1
		processor.diffBoth.Keys = appendStrArr(processor.diffBoth.Keys, 10, diff.Key)
	}
}

func appendStrArr(arr []string, size int, elem string) []string {
	if len(arr) >= size {
		return arr
	}
	return append(arr, elem)
}
