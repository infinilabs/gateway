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

package forcemerge

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"runtime"
	"time"
)

type ForceMergeModule struct {
}

func (this ForceMergeModule) Name() string {
	return "force_merge"
}

type Discovery struct {
	Enabled bool `config:"enabled"`
	MinIdleTime  string `config:"min_idle_time"`
	Interval string   `config:"interval"`
	Rules           []DiscoveryRule `config:"rules"`
}

type DiscoveryRule struct {
	IndexPattern string `config:"index_pattern"`
	TimeFields   []string `config:"timestamp_fields"`
}

type MergeConfig struct {
	Enabled         bool            `config:"enabled"`
	Elasticsearch   string          `config:"elasticsearch"`
	Indices         []string        `config:"indices"`
	MinSegmentCount int             `config:"min_num_segments"`
	MaxSegmentCount int             `config:"max_num_segments"`
	Discovery Discovery  			`config:"discovery"`

}

var mergeConfig = MergeConfig{}

func (module ForceMergeModule) Setup(cfg *config.Config) {

	ok, err := env.ParseConfig("force_merge", &mergeConfig)
	if ok && err != nil {
		panic(err)
	}

}

type ForceMergeTaskItem struct {
	Elasticsearch string
	Index string
}

func forceMerge(client elastic.API, index string) {

	retry := 0
	GET_STATS:
	stats, err := client.GetIndexStats(index)
	log.Debug(stats)
	if err != nil {
		log.Errorf("index [%v] error on get index stats, retry, %v", index, err)
		time.Sleep(60 * time.Second)
		retry++
		if retry > 120 {
			log.Errorf("retried 120 times, %v", err)
			return
		}
		goto GET_STATS
	}

FORCE_MERGE:
	if (stats.All.Primary.Segments.Count) > mergeConfig.MinSegmentCount && stats.All.Primary.Merges.Current == 0 {
		log.Infof("index [%v] has [%v] segments, going to do force_merge", index, stats.All.Primary.Segments.Count)
		err := client.Forcemerge(index, mergeConfig.MaxSegmentCount)
		if err != nil {
			log.Error(err)
			//TODO assume operation is send
			time.Sleep(60 * time.Second)
			retry++
			if retry > 120 {
				log.Errorf("retried 120 times, %v", err)
				return
			}
			goto GET_STATS
		}
	} else if stats.All.Primary.Segments.Count == 0 && stats.All.Primary.Store.SizeInBytes == 0 {
		log.Infof("error on get stats, index [%v] only has 0 segments, retry, %v", index, stats)
		ok, err := client.IndexExists(index)
		if err != nil {
			log.Error(err)
		}
		if !ok {
			log.Error("index not exists, ignore, ", index)
			return
		}

		time.Sleep(60 * time.Second)
		retry++
		if retry > 120 {
			log.Errorf("retried 120 times, %v", err)
			return
		}
		goto GET_STATS
	} else if stats.All.Primary.Merges.Current > 0 {
		log.Infof("index [%v] has [%v] segments, are still merging", index, stats.All.Primary.Segments.Count)
	} else if stats.All.Primary.Segments.Count > mergeConfig.MinSegmentCount {
		log.Infof("index [%v] has [%v] segments, are still merging", index, stats.All.Primary.Segments.Count)
	} else {
		log.Infof("index [%v] only has [%v] segments, skip force_merge", index, stats.All.Primary.Segments.Count)
		return
	}

	//let's wait
	time.Sleep(10 * time.Second)
	waitTime := time.Now().Add(2 * time.Hour)
WAIT_MERGE:

	if time.Now().After(waitTime) {
		log.Warn("wait [%v] too long, go for next index", index)
		return
	}

	stats, err = client.GetIndexStats(index)
	log.Debug(stats)
	if err != nil {
		log.Error(err)
		if util.ContainStr(err.Error(), "Timeout") {
			log.Error("wait 30s and try again.")
			time.Sleep(30 * time.Second)
			retry++
			goto WAIT_MERGE
		} else {
			log.Error("wait 60s and try again.")
			time.Sleep(60 * time.Second)
			retry++
			goto WAIT_MERGE
		}
		//continue
		//TODO
	}

	if stats.All.Primary.Segments.Count > mergeConfig.MaxSegmentCount+50 {
		//TODO, merge is not started
		time.Sleep(60 * time.Second)
		retry++
		if retry > 120 {
			return
		}
		goto FORCE_MERGE
	}

	if stats.All.Primary.Merges.Current > 0 {
		log.Infof("index %v still have %v merges are running.", index, stats.All.Primary.Merges.Current)
		if stats.All.Primary.Merges.Current > 10 {
			time.Sleep(60 * time.Second)
		} else {
			time.Sleep(10 * time.Second)
		}
		retry++
		if retry > 120 {
			return
		}
		goto GET_STATS
	} else {
		log.Infof("index %v has finished the force_merge, continue.", index)
	}
}

func (module ForceMergeModule) Start() error {

	if !mergeConfig.Enabled {
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
					log.Error("error in force_merge service", v)
				}
			}
		}()
		client := elastic.GetClient(mergeConfig.Elasticsearch)
		for i, v := range mergeConfig.Indices {
			log.Infof("#%v - start forcemerging index [%v]", i, v)
			forceMerge(client,v)
		}

		for {

			bytes,err:=queue.Pop(queue.GetOrInitConfig(taskQueue))
			if err!=nil{
				panic(err)
			}

			taskItem:=ForceMergeTaskItem{}
			util.FromJSONBytes(bytes,&taskItem)
			client := elastic.GetClient(mergeConfig.Elasticsearch)
			forceMerge(client,taskItem.Index)
		}

	}()

	if mergeConfig.Discovery.Enabled{
		task1 := task.ScheduleTask{
			Description: "discovery indices for force_merge",
			Type:        "interval",
			Interval:    "60m",
			Task: func() {
				client := elastic.GetClient(mergeConfig.Elasticsearch)
				for _,v:=range mergeConfig.Discovery.Rules{
					log.Trace("processing index_pattern: ",v.IndexPattern)
					indices,err:=client.GetIndices(v.IndexPattern)
					if err!=nil{
						panic(err)
					}
					if indices!=nil{
						for _,v:=range (*indices){
							if v.SegmentsCount> int64(mergeConfig.MinSegmentCount){
								task:=ForceMergeTaskItem{Elasticsearch: mergeConfig.Elasticsearch,Index: v.Index}
								log.Trace("add force_merge task to queue,",task)
								err:=queue.Push(queue.GetOrInitConfig(taskQueue),util.MustToJSONBytes(task))
								if err!=nil{
									panic(err)
								}
							}
						}
					}
				}
			},
		}
		task.RegisterScheduleTask(task1)
	}

	return nil
}

const taskQueue ="force_merge_tasks"

func (module ForceMergeModule) Stop() error {

	return nil
}
