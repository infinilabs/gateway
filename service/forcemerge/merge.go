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
	"runtime"
	"time"
)

type ForceMergeModule struct {
}

func (this ForceMergeModule) Name() string {
	return "forcemerge"
}

type MergeConfig struct {
	Enabled bool  `config:"enabled"`
	Elasticsearch string  `config:"elasticsearch"`
	Indices []string  `config:"indices"`
	MinSegmentCount int  `config:"min_num_segments"`
	MaxSegmentCount int  `config:"max_num_segments"`
}

var mergeConfig = MergeConfig{}
func (module ForceMergeModule) Setup(cfg *config.Config) {

	ok,err:=env.ParseConfig("forcemerge",&mergeConfig)
	if ok&&err!=nil{
		panic(err)
	}

}
func (module ForceMergeModule) Start() error {

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
					log.Error("error in forcemerge service", v)
				}
			}
		}()
		client:=elastic.GetClient(mergeConfig.Elasticsearch)
		for i,v:=range mergeConfig.Indices{
			log.Infof("#%v - forcemerging index [%v]",i,v)
			stats,err:=client.GetIndexStats(v)
			log.Debug(stats)
			if err!=nil{
				log.Error(err)
				continue
			}

			if stats.All.Primary.Segments.Count>mergeConfig.MinSegmentCount{
				log.Infof("index [%v] has [%v] segments, going to do forcemerge",v,stats.All.Primary.Segments.Count)
				err:=client.Forcemerge(v,mergeConfig.MaxSegmentCount)
				if err!=nil{
					log.Error(err)
					continue
				}
			}else{
				log.Infof("index [%v] only has [%v] segments, skip forcemerge",v,stats.All.Primary.Segments.Count)
				continue
			}

			//let's wait
			time.Sleep(10 * time.Second)
		WAIT_MERGE:
			stats,err=client.GetIndexStats(v)
			log.Debug(stats)
			if err!=nil{
				log.Error(err)
				continue
			}

			if stats.All.Primary.Merges.Current>0{
				log.Infof("index %v still have %v merges is running.",v,stats.All.Primary.Merges.Current)
				if stats.All.Primary.Merges.Current>10{
					time.Sleep(60 * time.Second)
				}else{
					time.Sleep(10 * time.Second)
				}
				goto WAIT_MERGE
			}else{
				log.Infof("index %v has finished the merges, continue.",v)
			}

		}
	}()

	return nil
}

func (module ForceMergeModule) Stop() error {

	return nil
}

