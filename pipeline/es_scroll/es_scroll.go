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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ScrollJoint struct {
	param.Parameters
	totalSize   int
	successSize int
	failureSize int
	batchSize int
	persist bool
	outputQueueName string
	esconfig elastic.ElasticsearchConfig
}

func (joint ScrollJoint) Name() string {
	return "es_scroll"
}

func (joint ScrollJoint) Open() error{
	sliceSizeVal, _ := joint.GetInt("slice_size", 10)
	joint.batchSize, _ = joint.GetInt("batch_size", 5000)
	fieldsVal, _ := joint.GetString("fields")
	scrollTimeVal := joint.GetStringOrDefault("scroll_time", "5m")
	queryVal := joint.GetStringOrDefault("query", "")
	indexNameVal := joint.GetStringOrDefault("indices", "filebeat-*")
	esNameVal := joint.GetStringOrDefault("elasticsearch", "default")
	joint.outputQueueName = joint.GetStringOrDefault("output_queue", "default")
	//dumpFile := joint.GetStringOrDefault("dump_file", "/tmp/translog.bin")
	joint.persist=joint.GetBool("persist",true)

	start := time.Now()
	//pipelines.Open(dumpFile)

	//TODO, 如果索引包含*，则自动展开成单独的索引任务，方便排错

	joint.esconfig=elastic.GetConfig(esNameVal)

	client := elastic.GetClient(esNameVal)

	wg := sync.WaitGroup{}

	if sliceSizeVal < 1 || client.GetMajorVersion() < 5 {
		log.Warnf("invalid slice config(%v) or not supported by elasticsearch(v%v)", sliceSizeVal, client.ClusterVersion())
		sliceSizeVal = 1
	}

	bars := []*pb.ProgressBar{}

	for slice := 0; slice < sliceSizeVal; slice++ {

		//log.Trace("slice, ", slice)

		tempSlice := slice

		scrollID, totalDocs, err := joint.NewScroll(indexNameVal, scrollTimeVal, joint.batchSize, queryVal, tempSlice, sliceSizeVal, fieldsVal)

		if err != nil {
			log.Debug(err)
			continue
		}

		log.Debugf("slice %v docs: %v", tempSlice, totalDocs)

		joint.totalSize += totalDocs

		//joint.ProcessScrollResult(result)

		if totalDocs == 0 {
			log.Tracef("slice %v is empty", tempSlice)
			continue
		}

		bar := pb.New(totalDocs).Prefix(fmt.Sprintf("Scroll#%v", slice))
		bars = append(bars, bar)

		wg.Add(1)

		go func() {
			var done bool
			// loop scrolling until done
			for scrollID = scrollID; done == false; done, scrollID, joint.batchSize = joint.Next(client, scrollID, scrollTimeVal) {
				bar.Add(joint.batchSize)
			}
			log.Tracef("slice %v is done", tempSlice)
			wg.Done()
			bar.Finish()
		}()

	}

	log.Debug("total docs: ", joint.totalSize)

	pool, err := pb.StartPool(bars...)
	if err != nil {
		panic(err)
	}
	wg.Wait()
	pool.Stop()

	duration := time.Now().Sub(start).Seconds()

	log.Infof("scroll finished, docs: %v, duration: %vs, qps: %v ", joint.totalSize, duration, math.Ceil(float64(joint.totalSize)/math.Ceil((duration))))

	return nil
}

func (joint ScrollJoint) Close() error{

	//pipelines.Flush()
	//pipelines.Sync()
	//pipelines.Close()

	return nil
}

func (joint ScrollJoint) Read() ([]byte, error){

	return nil,nil
}


func (joint ScrollJoint) Process(c *pipeline.Context) error {


	return nil
}

func BasicAuth(req *fasthttp.Request, user, pass string) {
	msg := fmt.Sprintf("%s:%s", user, pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(msg))
	req.Header.Add("Authorization", "Basic "+encoded)
}

func (joint ScrollJoint) NewScroll(indexNames string, scrollTime string, docBufferCount int, query string, slicedId, maxSlicedCount int, fields string) (scrollID string, totalDocs int, err error) {

	url := fmt.Sprintf("%s/%s/_search?scroll=%s&size=%d", joint.esconfig.Endpoint, indexNames, scrollTime, docBufferCount)
	var jsonBody []byte
	if len(query) > 0 || maxSlicedCount > 0 || len(fields) > 0 {
		queryBody := map[string]interface{}{}

		if len(fields) > 0 {
			if !strings.Contains(fields, ",") {
				log.Error("The fields shoud be seraprated by ,")
				return "", 0, errors.New("")
			} else {
				queryBody["_source"] = strings.Split(fields, ",")
			}
		}

		if len(query) > 0 {
			queryBody["query"] = map[string]interface{}{}
			queryBody["query"].(map[string]interface{})["query_string"] = map[string]interface{}{}
			queryBody["query"].(map[string]interface{})["query_string"].(map[string]interface{})["query"] = query
		}

		if maxSlicedCount > 1 {
			log.Tracef("sliced scroll, %d of %d", slicedId, maxSlicedCount)
			queryBody["slice"] = map[string]interface{}{}
			queryBody["slice"].(map[string]interface{})["id"] = slicedId
			queryBody["slice"].(map[string]interface{})["max"] = maxSlicedCount
		}

		jsonArray, err := json.Marshal(queryBody)
		if err != nil {
			log.Error(err)

		} else {
			jsonBody = jsonArray
		}
	}

	if jsonBody == nil {
		panic("scroll request is nil")
	}

	client := &fasthttp.Client{
		MaxConnsPerHost: 1000,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	BasicAuth(req, joint.esconfig.BasicAuth.Username, joint.esconfig.BasicAuth.Password)

	req.Header.SetContentType("application/json")
	req.Header.SetMethod(fasthttp.MethodGet)

	req.SetRequestURI(url)
	req.SetBody(jsonBody)

	err = client.Do(req, res)
	if err != nil {
		panic(err)
	}

	data := res.Body()

	hit, str1 := util.ExtractFieldFromJson(&data, []byte("\"_scroll_id\":\""), []byte("\","), []byte("_scroll_id"))

	scrollId := string(str1)
	if !hit {
		panic(errors.New("scroll_id parsed failed " + string(data)))
	}

	//check if scroll is over
	if util.IsBytesEndingWith(&data, []byte("\"hits\":[]}}")) {
		return scrollId, 0, errors.New("no docs returned")
	}

	hit, totalDocsStr := util.ExtractFieldFromJson(&data, []byte("hits\":{\"total\":{\"value\":"), []byte(",\"relation\":\""), []byte("\"total\":{\"value\""))
	i, err := strconv.Atoi(string(totalDocsStr))
	if err != nil {
		panic(errors.New("invalid total size, parsed faile"))
	}

	if joint.persist{
		joint.WriteContent(data)
		//queue.Push("scroll_results",data)
	}

	return scrollId, i, nil

}

func (joint ScrollJoint) NextScroll(scrollTime string, scrollId string) (string, error) {
	url := fmt.Sprintf("%s/_search/scroll?scroll=%s&scroll_id=%s", joint.esconfig.Endpoint, scrollTime, scrollId)

	client := &fasthttp.Client{
		MaxConnsPerHost: 1000,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	BasicAuth(req, joint.esconfig.BasicAuth.Username, joint.esconfig.BasicAuth.Password)

	req.Header.SetMethod(fasthttp.MethodGet)

	req.SetRequestURI(url)

	err := client.Do(req, res)
	if err != nil {
		panic(err)
	}

	data := res.Body()

	//fmt.Println(string(data))

	//check if scroll is over
	if util.IsBytesEndingWith(&data, []byte("\"hits\":[]}}")) {
		return scrollId, errors.New("no docs returned")
	}

	hit, str1 := util.ExtractFieldFromJson(&data, []byte("\"_scroll_id\":\""), []byte("\","), []byte("_scroll_id"))

	scrollId = string(str1)
	if !hit {
		panic(errors.New("scroll_id parsed failed " + string(data)))
	}

	if joint.persist{
		joint.WriteContent(data)
		//queue.Push("scroll_results",data)
	}

	return scrollId, nil
}

func (joint ScrollJoint) WriteContent(data []byte) {
	stats.Increment("translog","write")
	stats.IncrementBy("translog","bytes_write", int64(len(data)))

	//qName := "item-queue"
	//qDir := "/tmp"
	//segmentSize := 50

	//// Create a new queue with segment size of 50
	//q, err := dque.NewOrOpen(qName, qDir, segmentSize, ItemBuilder)
	//if err != nil {
	//	log.Fatal("Error creating new dque ", err)
	//}
	//
	//// Add an item to the queue
	//if err := q.Enqueue(&Item{"Joe", 1}); err != nil {
	//	log.Fatal("Error enqueueing item ", err)
	//}
	//log.Println("Size should be 1:", q.Size())
	//
	//// Properly close a queue
	//q.Close()


	queue.Push("scroll_results",data)

	return
}

func (joint ScrollJoint) Next(client elastic.API, scrollId string, scrollTime string) (bool, string, int) {

	nextScrollID, err := joint.NextScroll(scrollTime, scrollId)

	if err != nil {
		log.Debug(err)
		return true, nextScrollID, joint.batchSize
	}

	stats.IncrementBy("scroll", "total", int64(joint.batchSize))

	//joint.ProcessScrollResult(result)

	return false, nextScrollID, joint.batchSize
}

func (joint ScrollJoint) ProcessScrollResult(result elastic.ScrollResponseAPI) {

	stats.IncrementBy("scroll", "total", int64(len(result.GetDocs())))

	return

	//fmt.Println("hits total:",len(result.GetDocs()))

	//bar.Add(len(s.Hits.Docs))
	// show any failures
	//return

	for _, failure := range result.GetShardResponse().Failures {
		reason, _ := json.Marshal(failure.Reason)
		log.Errorf(string(reason))
		joint.failureSize++

		stats.Increment("scroll", "failure")
	}

	//successSize += len(result.GetDocs())

	// show any failures
	for _, failure := range result.GetShardResponse().Failures {
		reason, _ := json.Marshal(failure.Reason)
		log.Errorf(string(reason))
	}

	//data:=util.MustToJSONBytes(result.GetDocs())
	//WriteContent(&data)
	//outputQueueName := joint.GetStringOrDefault("output_queue", "es_queue")
	//
	//// write all the docs into a channel
	//for _, docI := range result.GetDocs() {
	//	queue.Push(outputQueueName, util.MustToJSONBytes(docI))
	//	//	joint.DocChan <- docI.(map[string]interface{})
	//}
}
