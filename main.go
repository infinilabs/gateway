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

package main

import (
	_ "expvar"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	queue2 "infini.sh/framework/modules/disk_queue"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/filter"
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/redis"
	"infini.sh/framework/modules/s3"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	_ "infini.sh/framework/plugins"
	stats "infini.sh/framework/plugins/stats_statsd"
	"infini.sh/gateway/config"
	_ "infini.sh/gateway/pipeline"
	"infini.sh/gateway/proxy"
	"infini.sh/gateway/service/floating_ip"
	"infini.sh/gateway/service/forcemerge"
	"infini.sh/gateway/service/translog"
)

func main() {

	terminalHeader := ("\n   ___   _   _____  __  __    __  _       \n")
	terminalHeader += ("  / _ \\ /_\\ /__   \\/__\\/ / /\\ \\ \\/_\\ /\\_/\\\n")
	terminalHeader += (" / /_\\///_\\\\  / /\\/_\\  \\ \\/  \\/ //_\\\\\\_ _/\n")
	terminalHeader += ("/ /_\\\\/  _  \\/ / //__   \\  /\\  /  _  \\/ \\ \n")
	terminalHeader += ("\\____/\\_/ \\_/\\/  \\__/    \\/  \\/\\_/ \\_/\\_/ \n\n")

	terminalFooter := ""

	app := framework.NewApp("gateway", "A light-weight, powerful and high-performance elasticsearch gateway.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	if app.Setup(func() {

		//load core modules first

		module.RegisterSystemModule(&stats2.SimpleStatsModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})

		module.RegisterUserPlugin(translog.TranslogModule{})
		module.RegisterSystemModule(&filter.FilterModule{})

		module.RegisterSystemModule(&s3.S3Module{})

		module.RegisterSystemModule(&queue2.DiskQueue{})
		module.RegisterSystemModule(&redis.RedisModule{})
		module.RegisterSystemModule(&elastic.ElasticModule{})

		module.RegisterSystemModule(&task.TaskModule{})
		module.RegisterUserPlugin(&proxy.GatewayModule{})
		module.RegisterUserPlugin(forcemerge.ForceMergeModule{})
		module.RegisterSystemModule(&pipeline.PipeModule{})
		module.RegisterUserPlugin(floating_ip.FloatingIPPlugin{})
		module.RegisterSystemModule(&api.APIModule{})

	}, func() {
		//start each module, with enabled provider
		module.Start()

	},nil){
		app.Run()
	}


}
