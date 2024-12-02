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

// 启动，如果是 active 模式，且没有存在虚拟节点，则切换为 standby 模式；
// 启动，如果是 standby 模式，如果没有存在虚拟节点，则切换为 active 模式；
// 运行中，active 节点开启心跳服务端，每 5s 广播 arp 地址；
// 运行中，standby 节点，连接虚拟节点访问 active 服务器，如果连接成功，继续检测
// 运行中，standby 节点，连接虚拟节点，如果连接失败，重试 3 次，则提升自己为 active 节点，执行 active 运行任务；
package floating_ip

import (
	"context"
	"math/rand"
	net1 "net"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"github.com/j-keck/arping"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/wrapper/net"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/service/heartbeat"
)

type EchoConfig struct {
	EchoPort        int `config:"port"`               //61111
	EchoDialTimeout int `config:"dial_timeout_in_ms"` //10s
	EchoTimeout     int `config:"timeout_in_ms"`      //10s
}

type FloatingIPConfig struct {
	Enabled   bool   `config:"enabled"`
	IP        string `config:"ip"`
	Netmask   string `config:"netmask"`
	Interface string `config:"interface"`

	LocalIP string `config:"local_ip"` //local ip address
	PeerIP  string `config:"peer_ip"`  //remote ip to failover

	Echo EchoConfig `config:"echo"`

	Priority               int  `config:"priority"`
	ForcedSwitchByPriority bool `config:"forced_by_priority"`

	BroadcastConfig config.NetworkConfig `config:"broadcast"`
}

var atomicActiveOrNot atomic.Value

type FloatingIPPlugin struct {
}

func (this FloatingIPPlugin) Name() string {
	return "floating_ip"
}

var (
	floatingIPConfig = FloatingIPConfig{
		Enabled: false,
		Netmask: "255.255.255.0",
		Echo: EchoConfig{
			EchoPort:        61111,
			EchoTimeout:     10000,
			EchoDialTimeout: 10000,
		},
		BroadcastConfig: config.NetworkConfig{
			Binding: "224.3.2.2:7654",
		},
	}
)

func (module FloatingIPPlugin) Setup() {
	ok, err := env.ParseConfig("floating_ip", &floatingIPConfig)
	if ok && err != nil  &&global.Env().SystemConfig.Configs.PanicOnConfigError{
		panic(err)
	}

	if !floatingIPConfig.Enabled {
		return
	}

	if !util.IsRootUser() {
		log.Error("floating_ip need to run as root user")
		floatingIPConfig.Enabled = false
		return
	}

	if floatingIPConfig.Interface == "" || floatingIPConfig.IP == "" || floatingIPConfig.LocalIP == "" {
		//let's do some magic
		dev, ip, mask, err := util.GetPublishNetworkDeviceInfo("")
		if err != nil {
			panic(err)
		}

		floatingIPConfig.LocalIP = ip

		if floatingIPConfig.Interface == "" {
			floatingIPConfig.Interface = dev
		}

		log.Tracef("local publish address: %v,%v,%v", dev, ip, mask)

		//if mask is not setting, try guess
		if floatingIPConfig.Netmask == "" {
			floatingIPConfig.Netmask = mask
		}

		if floatingIPConfig.IP == "" {
			prefix := util.GetIPPrefix(ip)
			floatingIPConfig.IP = prefix + ".234"
		}

		log.Debugf("try to use floating ip address: %v,%v,%v", dev, floatingIPConfig.IP, mask)
	}

	if floatingIPConfig.IP == "" || floatingIPConfig.Interface == "" || floatingIPConfig.LocalIP == "" {
		panic("invalid floating_ip config")
	}

	if floatingIPConfig.Priority < 1 {
		floatingIPConfig.Priority = rand.Intn(1000)
	}
}

var pingTimeout = []string{"timeout", "Unreachable", "unreachable"}
var pingAlive = []string{"ttl", "time="}

func pingActiveNode(ip string) bool {
	//TODO, 禁 ping 了，但是端口可以连接

	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ping", ip, "-i 1").Output()
	if err != nil {
		log.Debug(err, util.UnsafeBytesToString(out))
	}

	if util.ContainsAnyInArray(string(out), pingTimeout) {
		return false
	} else if util.ContainsAnyInArray(string(out), pingAlive) {
		return true
	} else {
		return false
	}
}

var srvSignal = make(chan bool, 10)
var arpSignal = make(chan bool, 10)
var multicastSignal = make(chan bool, 10)
var haCheckSignal = make(chan bool, 10)

func (module FloatingIPPlugin) SwitchToActiveMode() {

	if ok1, hit := atomicActiveOrNot.Load().(bool); hit && ok1 {
		log.Tracef("already in active mode, skip")
		return
	}

	atomicActiveOrNot.Store(true)

	log.Debugf("active floating_ip at: %v", floatingIPConfig.IP)

	err := net.SetupAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
	if err != nil {
		panic(err)
	}

	log.Tracef("floating_ip echo service :%v is up and running.", floatingIPConfig.Echo.EchoPort)

	//announce floating_ip, do arping every 10s
	task.RunWithinGroup("arping", func(ctx context.Context) error {
		for {
			select {
			case quit := <-arpSignal:
				if quit {
					log.Tracef("quit arping")
					return nil
				}
			default:
				if ok1, hit := atomicActiveOrNot.Load().(bool); hit && !ok1 {
					log.Tracef("not active, quit broadcast")
					return nil
				}

				log.Trace("announce floating_ip, do arping every 10s")
				ip := net1.ParseIP(floatingIPConfig.IP)
				err := arping.GratuitousArpOverIfaceByName(ip, floatingIPConfig.Interface)
				if err != nil {
					if util.ContainStr(err.Error(), "unable to open") {
						panic("please make sure running as root user, or sudo")
					}
					panic(err)
				}
				time.Sleep(10 * time.Second)
			}
		}
		return nil
	})

	//announce via broadcast
	task.RunWithinGroup("broadcast", func(ctx context.Context) error {
		req := Request{
			IsActive:   true,
			FloatingIP: floatingIPConfig.IP,
			FixedIP:    floatingIPConfig.LocalIP,
			EchoPort:   floatingIPConfig.Echo.EchoPort,
			Priority:   floatingIPConfig.Priority,
		}
		for {
			select {
			case quit := <-multicastSignal:
				if quit {
					log.Tracef("quit broadcast")
					return nil
				}
			default:
				if ok1, hit := atomicActiveOrNot.Load().(bool); hit && !ok1 {
					log.Tracef("not active, quit broadcast")
					return nil
				}
				log.Trace("announce floating_ip, do broadcast every 10s")
				Broadcast(&floatingIPConfig, &req)
				time.Sleep(10 * time.Second)
			}
		}
		return nil
	})

	actived = true
	log.Infof("floating_ip listen at: %v, %v, %v", floatingIPConfig.IP, floatingIPConfig.Echo.EchoPort, floatingIPConfig.Priority)
}

func (module FloatingIPPlugin) Deactivate(silence bool) {
	if actived || silence {
		log.Debugf("deactivating floating_ip at: %v", floatingIPConfig.IP)
		err := net.DisableAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
		if err != nil && !silence {
			log.Error(err)
		}

		if actived {
			srvSignal <- true
			multicastSignal <- true
			arpSignal <- true
		}

		log.Tracef("floating_ip at: %v deactivated", floatingIPConfig.IP)
	}
	actived = false
}

func (module FloatingIPPlugin) SwitchToStandbyMode(latency time.Duration) {
	if ok1, hit := atomicActiveOrNot.Load().(bool); hit && !ok1 {
		log.Tracef("already in standby mode, skip")
		return
	}

	atomicActiveOrNot.Store(false)

	module.Deactivate(false)

	log.Infof("floating_ip entering standby mode")

	if latency > 0 {
		time.Sleep(latency)
	}

	task.RunWithinGroup("standby", func(ctx context.Context) error {
		aliveChan := make(chan bool)
		client:=heartbeat.New()
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
						log.Error(v)
					}
				}
				aliveChan <- false
			}()
			log.Tracef("check floating_ip echo_port:%v", floatingIPConfig.Echo.EchoPort)
			client.Start(floatingIPConfig.IP, floatingIPConfig.Echo.EchoPort, floatingIPConfig.Echo.EchoDialTimeout, floatingIPConfig.Echo.EchoTimeout, func() {
				aliveChan <- true
			}, func() {
				aliveChan <- false
			})
		}()

	WAIT:
		alive := <-aliveChan
		if !alive {
			log.Debug("floating_ip is not responding, promoting self")
			client.Stop()
			module.SwitchToActiveMode()
		} else {
			goto WAIT
		}
		return nil
	})

}

var actived bool

func (module FloatingIPPlugin) Start() error {

	if !floatingIPConfig.Enabled {
		log.Trace("floating_ip disabled")
		return nil
	}

	log.Debugf("setup floating_ip, root privilege are required")

	if !util.HasSudoPermission() {
		return errors.New("root privilege are required to use floating_ip.")
	}

	//start heart server
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
					log.Trace("error on heartbeat server:", v)
				}
			}
		}()
		err := heartbeat.StartServer("0.0.0.0", floatingIPConfig.Echo.EchoPort)
		if err != nil {
			panic(err)
		}
	}()

	//start broadcast listener
	go ServeMulticastDiscovery(&floatingIPConfig, func(addr *net1.UDPAddr, n int, bytes []byte) {
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
					log.Trace("error on broadcast listener:", v)
				}
			}
		}()

		if !floatingIPConfig.ForcedSwitchByPriority && !actived {
			log.Tracef("i am standby, no bother multicast message")
			return
		}

		//我是 master，别人也是 master
		//我不是 master，忽略
		//我是 master，忽略
		v := Request{}

		util.MustFromJSONBytes(bytes[:n], &v)

		log.Tracef("received multicast message: %v", util.ToJson(v, false))

		if v.FixedIP == floatingIPConfig.LocalIP {
			log.Tracef("received my message: %v", util.ToJson(v, false))
			return
		} else {
			//active node
			if actived {
				if v.FixedIP != floatingIPConfig.LocalIP {
					log.Debugf("received another host declared as active: %v", util.ToJson(v, false))
					if v.Priority > floatingIPConfig.Priority {
						log.Tracef("received high priority message, switch to backup mode: %v", util.ToJson(v, false))
						module.SwitchToStandbyMode(1 * time.Second)
						return
					}
				}
			} else {
				//standby mode
				//local with higher priority
				if floatingIPConfig.Priority > v.Priority && floatingIPConfig.ForcedSwitchByPriority {
					//yelling, i am with higher priority
					req := Request{
						IsActive:   true,
						FloatingIP: floatingIPConfig.IP,
						FixedIP:    floatingIPConfig.LocalIP,
						EchoPort:   floatingIPConfig.Echo.EchoPort,
						Priority:   floatingIPConfig.Priority,
					}
					log.Infof("yo, i am with higher priority: %v", floatingIPConfig.Priority)
					module.SwitchToActiveMode()
					time.Sleep(5 * time.Second)
					Broadcast(&floatingIPConfig, &req)
				}
			}
		}
	})

	//stop previous unclean status
	module.Deactivate(true)

	module.StateMachine()

	return nil
}

type State string

const Active State = "Active"
const Backup State = "Backup"
const Candidate State = "Candidate"
const PreviousActiveIsBack State = "PreviousActiveIsBack"

func (module FloatingIPPlugin) StateMachine() {
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
				log.Error(v)
			}
		}
	}()

	client:=heartbeat.New()
	aliveChan := make(chan bool)
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
					log.Error(v)
				}
			}
		}()

		err := client.Start(floatingIPConfig.IP, floatingIPConfig.Echo.EchoPort, floatingIPConfig.Echo.EchoDialTimeout, floatingIPConfig.Echo.EchoTimeout, func() {
			aliveChan <- true
		}, func() {
			aliveChan <- false
		})
		if err != nil {
			aliveChan <- false
		}
	}()

	alive := <-aliveChan

	if !alive {

		client.Stop()

		time.Sleep(10 * time.Second)
		//target floating_ip can't connect, but ip ping is alive
		if util.TestTCPPort(floatingIPConfig.IP, floatingIPConfig.Echo.EchoPort, 10*time.Second) && pingActiveNode(floatingIPConfig.IP) {
			log.Warnf("the floating_ip [%v] has been taken by someone, but the echo_port [%v] is not responding, promoting self", floatingIPConfig.IP, floatingIPConfig.Echo.EchoPort)
			//try to take it back
			module.SwitchToActiveMode()
			return
		}
	}

	log.Tracef("active floating_ip node found: %v", alive)

	if alive {
		module.SwitchToStandbyMode(5 * time.Second)
	} else {
		module.SwitchToActiveMode()
	}
}

func (module FloatingIPPlugin) Stop() error {
	if !floatingIPConfig.Enabled {
		return nil
	}

	log.Tracef("stopping floating_ip module")

	if actived {
		module.Deactivate(false)
	} else {
		haCheckSignal <- true
	}
	log.Tracef("floating_ip module stopped")
	return nil
}
