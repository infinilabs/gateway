/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
	_ "expvar"
	log "github.com/cihub/seelog"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/metrics"
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/queue"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	"infini.sh/framework/modules/redis"
	"infini.sh/framework/modules/s3"
	"infini.sh/framework/modules/security"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	_ "infini.sh/framework/plugins"
	"infini.sh/framework/plugins/managed/client"
	stats "infini.sh/framework/plugins/stats_statsd"
	"infini.sh/gateway/config"
	_ "infini.sh/gateway/pipeline"
	"infini.sh/gateway/proxy"
	"infini.sh/gateway/service/floating_ip"
	"infini.sh/gateway/service/forcemerge"
)

func setup() {
	module.RegisterSystemModule(&stats2.SimpleStatsModule{})
	module.RegisterUserPlugin(&stats.StatsDModule{})
	module.RegisterSystemModule(&s3.S3Module{})
	module.RegisterSystemModule(&queue2.DiskQueue{})
	module.RegisterSystemModule(&redis.RedisModule{})
	module.RegisterSystemModule(&elastic.ElasticModule{})
	module.RegisterSystemModule(&queue.Module{})
	module.RegisterSystemModule(&security.Module{})
	module.RegisterSystemModule(&task.TaskModule{})
	module.RegisterSystemModule(&api.APIModule{})
	module.RegisterModuleWithPriority(&pipeline.PipeModule{},100)

	module.RegisterUserPlugin(forcemerge.ForceMergeModule{})
	module.RegisterUserPlugin(floating_ip.FloatingIPPlugin{})
	module.RegisterUserPlugin(&metrics.MetricsModule{})
	module.RegisterPluginWithPriority(&proxy.GatewayModule{},200)
}

func start() {
	module.Start()

	err:= client.ConnectToManager()
	if err!=nil{
		log.Warn(err)
	}

	err= client.ListenConfigChanges()
	if err!=nil{
		log.Warn(err)
	}

}

func main() {

	terminalHeader := ("\n   ___   _   _____  __  __    __  _       \n")
	terminalHeader += ("  / _ \\ /_\\ /__   \\/__\\/ / /\\ \\ \\/_\\ /\\_/\\\n")
	terminalHeader += (" / /_\\///_\\\\  / /\\/_\\  \\ \\/  \\/ //_\\\\\\_ _/\n")
	terminalHeader += ("/ /_\\\\/  _  \\/ / //__   \\  /\\  /  _  \\/ \\ \n")
	terminalHeader += ("\\____/\\_/ \\_/\\/  \\__/    \\/  \\/\\_/ \\_/\\_/ \n\n")

	terminalFooter := ""

	app := framework.NewApp("gateway", "A light-weight, powerful and high-performance search gateway.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.BuildNumber), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	if app.Setup(setup, start, nil) {
		app.Run()
	}

}
