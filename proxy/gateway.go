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

	NotifyOnConfigSectionChange("elasticsearch", func(pCfg, cCfg *Config) {

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
					log.Error("error on apply elasticsearch change,", v)
				}
			}
		}()

		if cCfg != nil {
			// The framework replaces the cluster metadata with fresh objects
			// on elasticsearch config reload, while cached flows hold filters
			// that captured the old metadata pointers. Drop the cached flows
			// and re-resolve the entry flow handlers so filters rebuild
			// against the fresh metadata, the same way flow changes are
			// applied. Registered after the framework's own elasticsearch
			// reload callback (system module starts first), so the new
			// metadata is already in place when flows rebuild.
			common.ClearFlowCaches()
			for _, v := range module.entryPoints {
				v.RefreshDefaultFlow()
				v.RefreshTracingFlow()
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

			// Build the desired entrypoint set from the new config: entries
			// identical to a running one are kept untouched (skipKeys), only
			// changed or new entries get a fresh Entrypoint.
			old := module.entryPoints
			// ids whose config is unchanged, old entrypoint is kept as-is
			skipKeys := map[string]struct{}{}
			// ids with changed or new config, require a fresh entrypoint
			entryPoints := map[string]*entry.Entrypoint{}

			for _, v := range newConfig {
				if v.ID == "" && v.Name != "" {
					v.ID = v.Name
				}

				oldC, ok := old[v.ID]
				if ok {
					config := oldC.GetConfig()
					if config.Equals(&v) {
						skipKeys[v.ID] = struct{}{}
						continue
					}
				}

				applyDefaultReusePort(&v, module.DisableReusePortByDefault)
				e := entry.NewEntrypoint(v)
				entryPoints[v.ID] = e
			}

			// This file change did not add new entries or modify existing entries, nothing to do.
			if len(entryPoints) == 0 {
				return
			}

			// Switching to the new entrypoints runs in three phases. The new
			// entries fall into two groups by their reuse_port setting:
			// reuse_port=false entries cannot bind while the old listener is
			// still running ("address already in use"), so their predecessors
			// must be stopped first; reuse_port=true entries rely on
			// SO_REUSEPORT to bind alongside the old listener, which enables a
			// zero-downtime start-then-stop switch.
			//
			// 1. Stop the old entrypoints whose replacement cannot reuse the port.
			stoppedOld := map[string]*entry.Entrypoint{}
			for id, e := range entryPoints {
				if e.GetConfig().NetworkConfig.ReusePortEnabled() {
					continue
				}
				if oldC, ok := old[id]; ok {
					log.Trace("stopping ", oldC.GetNameOrID())
					oldC.Stop()
					stoppedOld[id] = oldC
				}
			}

			// 2. Start all new entrypoints. Each start is isolated so one
			// failure does not abort the others; when the old entrypoint was
			// stopped in phase 1, roll back to it so the port keeps serving
			// the old config.
			log.Debug("starting new entry points")
			for id, e := range entryPoints {
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error("error on start entry ", e.GetNameOrID(), ", ", r)
							delete(entryPoints, id)
							if oldC, ok := stoppedOld[id]; ok {
								log.Debug("rollback to old entry: ", oldC.GetNameOrID())
								if err := oldC.Start(); err != nil {
									log.Error("error on rollback entry ", oldC.GetNameOrID(), ", ", err)
								} else {
									entryPoints[id] = oldC
									delete(stoppedOld, id)
								}
							}
						}
					}()
					e.Start()
				}()
			}

			// 3. Stop the remaining old entrypoints: those replaced by
			// reuse_port=true entries and those no longer present in the new
			// config. Unchanged entries (skipKeys) and entries already stopped
			// in phase 1 are skipped.
			log.Debug("stopping old entry points")
			for id, v := range old {
				_, ok := skipKeys[id]
				if ok {
					entryPoints[id] = v
					continue
				}
				if _, ok := stoppedOld[id]; ok {
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
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	ok, err = env.ParseConfig("entry", &entryConfigs)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	log.Trace(util.ToJson(entryConfigs, true))

	ok, err = env.ParseConfig("flow", &flowConfigs)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
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
		applyDefaultReusePort(&v, module.DisableReusePortByDefault)
		e := entry.NewEntrypoint(v)
		if v.ID == "" && v.Name != "" {
			v.ID = v.Name
		}
		entryPoints[v.ID] = e
	}
	return entryPoints
}

// Helper function to fill in the default reuse_port for entries that do not
// configure it explicitly, leaving per-entry values untouched.
func applyDefaultReusePort(cfg *common.EntryConfig, disableReuseByDefault bool) {
	if cfg.NetworkConfig.ReusePort == nil {
		reuse := !disableReuseByDefault
		cfg.NetworkConfig.ReusePort = &reuse
	}
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
