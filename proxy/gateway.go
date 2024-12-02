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

package proxy

import (
	"runtime"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	api2 "infini.sh/gateway/api"
	"infini.sh/gateway/common"
	"infini.sh/gateway/proxy/entry"
)

type GatewayModule struct {
	api.Handler

	entryPoints map[string]*entry.Entrypoint

	API struct {
		Enabled bool `config:"enabled"`
	} `config:"api"`

	ORM struct {
		Enabled bool `config:"enabled"`
	} `config:"orm"`

	DisableReusePortByDefault bool `config:"disable_reuse_port_by_default"`
}

func (this *GatewayModule) Name() string {
	return "gateway"
}

func (module *GatewayModule) Setup() {

	module.entryPoints = module.loadEntryPoints()

	api := api2.GatewayAPI{}
	if module.API.Enabled {
		api.RegisterAPI("")
	}
	if module.ORM.Enabled {
		api.RegisterSchema()
	}

	module.registerAPI("")

}

func (module *GatewayModule) handleConfigureChange() {

	NotifyOnConfigSectionChange("flow", func(pCfg, cCfg *Config) {

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
					log.Error("error on apply flow change,", v)
				}
			}
		}()

		if cCfg != nil {
			//TODO diff previous and current config
			newConfig := []common.FlowConfig{}
			err := cCfg.Unpack(&newConfig)
			if err != nil {
				log.Error(err)
				return
			}

			for _, v := range newConfig {
				common.RegisterFlowConfig(v)
			}

			//just in case
			for _, v := range module.entryPoints {
				v.RefreshDefaultFlow()
				v.RefreshTracingFlow()
			}

			//修改完Flow，需要重启服务入口
			for _, v := range module.entryPoints {
				//TODO skip unnecessary restart
				log.Trace("stopping ", v.GetNameOrID())
				v.Stop()
				log.Trace("stopped ", v.GetNameOrID())
				v.Start()
				log.Trace("started ", v.GetNameOrID())

			}
		}
	})

	NotifyOnConfigSectionChange("router", func(pCfg, cCfg *Config) {
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
					log.Error("error on apply router change,", v)
				}
			}
		}()

		if cCfg != nil {
			newConfig := []common.RouterConfig{}
			err := cCfg.Unpack(&newConfig)
			if err != nil {
				log.Error(err)
				return
			}

			keys := map[string]string{}
			for _, v := range newConfig {
				if v.ID == "" && v.Name != "" {
					v.ID = v.Name
				}

				keys[v.ID] = v.ID
				common.RegisterRouterConfig(v)
			}

			//修改完路由，需要重启服务入口
			for _, v := range module.entryPoints {
				_, ok := keys[v.GetConfig().RouterConfigName]
				if ok {
					v.Stop()
					v.Start()
				}
			}
		}
	})

	NotifyOnConfigSectionChange("entry", func(pCfg, cCfg *Config) {

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
					log.Error("error on apply entry change,", v)
				}
			}
		}()

		if cCfg != nil {
			newConfig := []common.EntryConfig{}
			err := cCfg.Unpack(&newConfig)
			if err != nil {
				log.Error(err)
				return
			}

			//each entry should reuse port
			//collect old entry with same id and same port
			old := module.entryPoints
			existKeys := map[string]string{}
			skipKeys := map[string]string{}
			entryPoints := map[string]*entry.Entrypoint{}

			for _, v := range newConfig {

				if v.ID == "" && v.Name != "" {
					v.ID = v.Name
				}

				oldC, ok := old[v.ID]
				if ok {
					existKeys[v.ID] = v.ID
					config := oldC.GetConfig()
					if config.Equals(&v) {
						skipKeys[v.ID] = v.ID
						continue
					}
				}

				//if !module.DisableReusePortByDefault {
				//	v.NetworkConfig.ReusePort = true
				//}
				e := entry.NewEntrypoint(v)
				entryPoints[v.ID] = e
			}

			if len(entryPoints) == 0 {
				return
			}

			log.Debug("starting new entry points")
			for _, v := range entryPoints {
				v.Start()
			}

			log.Debug("stopping old entry points")
			for _, v := range old {
				_, ok := skipKeys[v.GetConfig().ID]
				if ok {
					entryPoints[v.GetConfig().ID] = v
					continue
				}
				v.Stop()
			}

			module.entryPoints = entryPoints
		}
	})

}

func (module *GatewayModule) loadEntryPoints() map[string]*entry.Entrypoint {

	routerConfigs := []common.RouterConfig{}
	flowConfigs := []common.FlowConfig{}
	entryConfigs := []common.EntryConfig{}

	ok, err := env.ParseConfig("gateway", &module)
	if ok && err != nil  &&global.Env().SystemConfig.Configs.PanicOnConfigError{
		panic(err)
	}

	ok, err = env.ParseConfig("entry", &entryConfigs)
	if ok && err != nil  &&global.Env().SystemConfig.Configs.PanicOnConfigError{
		panic(err)
	}

	log.Trace(util.ToJson(entryConfigs, true))

	ok, err = env.ParseConfig("flow", &flowConfigs)
	if ok && err != nil  &&global.Env().SystemConfig.Configs.PanicOnConfigError{
		panic(err)
	}

	if ok {
		for _, v := range flowConfigs {
			common.RegisterFlowConfig(v)
		}
	}

	ok, err = env.ParseConfig("router", &routerConfigs)
	if ok && err != nil {
		panic(err)
	}

	if ok {
		for _, v := range routerConfigs {
			common.RegisterRouterConfig(v)
		}
	}

	log.Trace("num of entry configs:", len(entryConfigs))
	entryPoints := map[string]*entry.Entrypoint{}
	for _, v := range entryConfigs {
		//if !module.DisableReusePortByDefault {
		//	v.NetworkConfig.ReusePort = true
		//}
		e := entry.NewEntrypoint(v)
		if v.ID == "" && v.Name != "" {
			v.ID = v.Name
		}
		entryPoints[v.ID] = e
	}
	return entryPoints
}

func (module *GatewayModule) Start() error {

	log.Trace("num of entry_points:", len(module.entryPoints))
	for _, v := range module.entryPoints {
		log.Trace("start entry:", v.String())
		err := v.Start()
		log.Trace("finished start entry:", v.String(), ",err:", err)

		if err != nil {
			panic(err)
		}
	}

	module.handleConfigureChange()

	return nil
}

func (module *GatewayModule) Stop() error {

	for _, v := range module.entryPoints {
		err := v.Stop()
		if err != nil {
			panic(err)
		}
	}

	return nil
}
