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
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
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

	for slice := 0; slice < sliceSizeVal; slice++ {

		tempSlice := slice
		scrollResponse, err := client.NewScroll(indexNameVal, scrollTimeVal, joint.batchSize, queryVal, tempSlice, sliceSizeVal, fieldsVal, sortField, sortType)
		if err != nil {
			log.Error(err)
			continue
		}

		scrollResponse1, ok := scrollResponse.(elastic.ScrollResponseAPI)
		if !ok {
			log.Warn("invalid scroll response, ",scrollResponse, err)
			break
		}

		log.Debug("total docs for scrolling: ",scrollResponse1.GetHitsTotal())

		docs := scrollResponse1.GetDocs()
		docSize := len(docs)
		//joint.totalSize += docSize
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
			var scrollResponse interface{}
			initScrollID := scrollResponse1.GetScrollId()

			for {
				scrollResponse, err = client.NextScroll(scrollTimeVal, initScrollID)
				obj, ok := scrollResponse.(elastic.ScrollResponseAPI)
				initScrollID = obj.GetScrollId()
				if !ok {
					log.Debug(scrollResponse, err)
					break
				}

				docs := obj.GetDocs()
				docSize := len(docs)
				//joint.totalSize += docSize
				if docSize == 0 {
					log.Trace(scrollResponse)
					break
				}

				processingDocs(docs, joint.outputQueueName)

			}
			log.Tracef("slice %v is done", tempSlice)
			wg.Done()
		}()

	}

	//log.Debug("total docs: ", joint.totalSize)

	wg.Wait()

	//duration := time.Now().Sub(start).Seconds()

	//log.Infof("scroll finished, docs: %v, duration: %vs, qps: %v ", joint.totalSize, duration, math.Ceil(float64(joint.totalSize)/math.Ceil((duration))))

	return nil
}

func processingDocs(docs []interface{}, outputQueueName string) {
	for _, v := range docs {
		err := queue.Push(outputQueueName, util.MustToJSONBytes(v))
		if err != nil {
			log.Error(err)
		}
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
