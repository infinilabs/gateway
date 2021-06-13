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
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"runtime"
)

type IndexDiffModule struct {
}

func (this IndexDiffModule) Name() string {
	return "index_diff"
}

type Config struct {
	Enabled bool `config:"enabled"`
	Source  struct {
		Elasticsearch string `config:"elasticsearch"`
		IndexPattern  string `config:"index_pattern"`
		SortField     string `config:"sort_field"`
		RangeFrom     string `config:"range_from"`
		RangeTo       string `config:"range_to"`
	} `config:"source"`

	Target struct {
		Elasticsearch string `config:"elasticsearch"`
		IndexPattern  string `config:"index_pattern"`
		SortField     string `config:"sort_field"`
		RangeFrom     string `config:"range_from"`
		RangeTo       string `config:"range_to"`
	} `config:"target"`
}

var diffConfig = Config{}

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

		//source := elastic.GetClient(diffConfig.Source.Elasticsearch)

	}()

	return nil
}

func (module IndexDiffModule) Stop() error {

	return nil
}
