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
	pipe "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/filter"
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/queue"
	"infini.sh/framework/modules/task"
	stats "infini.sh/framework/plugins/stats_statsd"
	api2 "infini.sh/gateway/api"
	"infini.sh/gateway/config"
	"infini.sh/gateway/pipeline/dump_hash"
	"infini.sh/gateway/pipeline/index_diff"
	"infini.sh/gateway/service/diskqueue_consumer"
	"infini.sh/gateway/service/floating_ip"
	"infini.sh/gateway/service/forcemerge"
	"infini.sh/gateway/service/gateway"
	"infini.sh/gateway/service/indexing"
	"infini.sh/gateway/service/offline_processing"
	"infini.sh/gateway/service/translog"
)

func main() {

	terminalHeader := ("   ___   _   _____  __  __    __  _       \n")
	terminalHeader += ("  / _ \\ /_\\ /__   \\/__\\/ / /\\ \\ \\/_\\ /\\_/\\\n")
	terminalHeader += (" / /_\\///_\\\\  / /\\/_\\  \\ \\/  \\/ //_\\\\\\_ _/\n")
	terminalHeader += ("/ /_\\\\/  _  \\/ / //__   \\  /\\  /  _  \\/ \\ \n")
	terminalHeader += ("\\____/\\_/ \\_/\\/  \\__/    \\/  \\/\\_/ \\_/\\_/ \n\n")

	terminalFooter := ("Thanks for using INFINI GATEWAY, have a good day!")

	app := framework.NewApp("gateway", "A light-weight, powerful and high-performance elasticsearch gateway.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	app.Start(func() {

		//load core modules first
		module.RegisterSystemModule(elastic.ElasticModule{})
		module.RegisterUserPlugin(translog.TranslogModule{})
		module.RegisterSystemModule(filter.FilterModule{})

		module.RegisterSystemModule(queue.DiskQueue{})
		module.RegisterSystemModule(&queue.RedisModule{})

		module.RegisterSystemModule(&queue.QueueModule{})

		module.RegisterSystemModule(api.APIModule{})
		module.RegisterSystemModule(pipeline.PipeModule{})
		module.RegisterSystemModule(task.TaskModule{})

		module.RegisterUserPlugin(stats.StatsDModule{})
		module.RegisterUserPlugin(gateway.GatewayModule{})
		module.RegisterUserPlugin(floating_ip.FloatingIPPlugin{})
		module.RegisterUserPlugin(forcemerge.ForceMergeModule{})

		api2.Init()

		//register pipeline joints
		pipe.RegisterPipeJoint(indexing.JsonIndexingJoint{})
		pipe.RegisterPipeJoint(indexing.BulkIndexingJoint{})
		pipe.RegisterPipeJoint(diskqueue_consumer.DiskQueueConsumer{})
		pipe.RegisterPipeJoint(offline_processing.FlowRunner{})

		//TODO auto register plugins
		pipe.RegisterPlugin("index_diff", index_diff.New)
		pipe.RegisterPlugin("dump_hash", scroll.NewDumpHashProcessor)

		//start each module, with enabled provider
		module.Start()

	}, func() {
	})

}
