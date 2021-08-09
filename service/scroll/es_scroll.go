/*
Copyright 2016 Medcl (m AT medcl.net)

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

package scroll

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/segmentio/fasthash/fnv1a"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/rotate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"path"
	"sync"
)

type ScrollJoint struct {
	param.Parameters
	//totalSize       int
	//successSize     int
	//failureSize     int
	batchSize       int
	persist         bool
	outputQueueName string
	esconfig        *elastic.ElasticsearchConfig
}

func (joint ScrollJoint) Name() string {
	return "es_scroll"
}


var scrollResponsePool = &sync.Pool{
	New: func() interface{} {
		c := elastic.ScrollResponse{}
		return &c
	},
}
var scrollResponseV7Pool = &sync.Pool{
	New: func() interface{} {
		c := elastic.ScrollResponseV7{}
		return &c
	},
}


func (joint ScrollJoint) Open() error {
	sliceSizeVal, _ := joint.GetInt("slice_size", 1)
	joint.batchSize, _ = joint.GetInt("batch_size", 5000)
	fieldsVal, _ := joint.GetString("fields")
	scrollTimeVal := joint.GetStringOrDefault("scroll_time", "5m")
	queryVal := joint.GetStringOrDefault("query", "")
	indexNameVal := joint.GetStringOrDefault("indices", "filebeat-*")
	sortField := joint.GetStringOrDefault("sort_field", "")
	sortType := joint.GetStringOrDefault("sort_type", "asc")
	esNameVal := joint.GetStringOrDefault("elasticsearch", "default")
	joint.outputQueueName = joint.GetStringOrDefault("output_queue", "default")
	joint.persist = joint.GetBool("persist", true)

	//start := time.Now()

	joint.esconfig = elastic.GetConfig(esNameVal)
	client := elastic.GetClient(esNameVal)
	wg := sync.WaitGroup{}

	if sliceSizeVal < 1 || client.GetMajorVersion() < 5 {
		sliceSizeVal = 1
	}

	var totalSize  int

	for slice := 0; slice < sliceSizeVal; slice++ {

		tempSlice := slice
		scrollResponse1, err := client.NewScroll(indexNameVal, scrollTimeVal, joint.batchSize, queryVal, tempSlice, sliceSizeVal, fieldsVal, sortField, sortType)
		if err != nil {
			log.Error(err)
			continue
		}

		docs := scrollResponse1.GetDocs()
		docSize := len(docs)
		totalSize += docSize
		if docSize > 0 {
			processingDocs(docs, joint.outputQueueName)
			//joint.totalSize += len(scrollResponse1.GetDocs())
		}

		log.Debugf("slice %v docs: %v", tempSlice, scrollResponse1.GetHitsTotal())

		if scrollResponse1.GetHitsTotal() == 0 {
			log.Tracef("slice %v is empty", tempSlice)
			continue
		}
		wg.Add(1)

		go func() {
			defer wg.Done()
			var scrollResponse interface{}
			initScrollID := scrollResponse1.GetScrollId()

			version:=client.GetMajorVersion()

			for {
				data, err := client.NextScroll(scrollTimeVal, initScrollID)

				if version>=7{
					scrollResponse= scrollResponseV7Pool.Get().(*elastic.ScrollResponseV7)

					err=scrollResponse.(*elastic.ScrollResponseV7).UnmarshalJSON(data)

					//var json = jsoniter.ConfigCompatibleWithStandardLibrary
					//err=json.Unmarshal(data,scrollResponse)

					//iter := jsoniter.ConfigFastest.BorrowIterator(data)
					//iter.ReadVal(scrollResponse)
					//if iter.Error != nil {
					//	fmt.Println("error:", iter.Error)
					//}
					//jsoniter.ConfigFastest.ReturnIterator(iter)

					//err=json.Unmarshal(data,scrollResponse)

					if err != nil {
						panic(err)
					}
				}else{
					scrollResponse= scrollResponsePool.Get().(*elastic.ScrollResponse)
					err=scrollResponse.(*elastic.ScrollResponse).UnmarshalJSON(data)

					//var json = jsoniter.ConfigCompatibleWithStandardLibrary
					//err=json.Unmarshal(data,scrollResponse)

					//iter := jsoniter.ConfigFastest.BorrowIterator(data)
					//iter.ReadVal(scrollResponse)
					//if iter.Error != nil {
					//	fmt.Println("error:", iter.Error)
					//}
					//jsoniter.ConfigFastest.ReturnIterator(iter)

					//err=json.Unmarshal(data,scrollResponse)

					if err != nil {
						panic(err)
					}
				}

				obj, ok := scrollResponse.(elastic.ScrollResponseAPI)
				if !ok {
					if err != nil {
						panic(err)
					}
					break
				}

				initScrollID = obj.GetScrollId()

				docs := obj.GetDocs()
				docSize := len(docs)
				totalSize += docSize

				stats.Gauge(fmt.Sprintf("scroll_total_received-%v",tempSlice),joint.outputQueueName, int64(totalSize))

				if docSize == 0 {
					log.Trace(scrollResponse)
					break
				}

				processingDocs(docs, joint.outputQueueName)

				if version>=7{
					scrollResponseV7Pool.Put(scrollResponse)
				}else{
					scrollResponsePool.Put(scrollResponse)
				}

			}
			log.Debugf("slice %v is done", slice)

		}()

	}

	log.Infof("scroll finished, docs: %v ", totalSize)

	wg.Wait()

	//duration := time.Now().Sub(start).Seconds()

	//log.Infof("scroll finished, docs: %v, duration: %vs, qps: %v ", totalSize, duration, math.Ceil(float64(joint.totalSize)/math.Ceil((duration))))

	return nil
}

func processingDocs(docs []elastic.IndexDocument, outputQueueName string) {


	for _, v := range docs {

		//bytes:=util.MustToJSONBytes(v)
		stats.Increment("scrolling_processing."+outputQueueName,"docs")

		h1 := fnv1a.HashBytes32(util.MustToJSONBytes(v.Source))
		//hash := util.MustToJSONBytes(h1)

		//fmt.Println(v.Index,",",v.ID,",",util.IntToString(int(h1)))

		handler:=rotate.GetFileHandler(path.Join(global.Env().GetDataDir(),"diff",outputQueueName),rotate.DefaultConfig)
		handler.WriteBytesArray([]byte((v.ID.(string))),[]byte(","),[]byte(util.IntToString(int(h1))),[]byte("\n"))
		//handler.Close()

		//err := queue.Push(outputQueueName, bytes)
		//if err != nil {
		//	log.Error(err)
		//}
	}
}

func (joint ScrollJoint) Close() error {
	return nil
}

func (joint ScrollJoint) Read() ([]byte, error) {
	return nil, nil
}

func (joint ScrollJoint) Process(c *pipeline.Context) error {

	return nil
}
