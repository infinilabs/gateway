//启动，如果是 active 模式，且没有存在虚拟节点，则切换为 standby 模式；
//启动，如果是 standby 模式，如果没有存在虚拟节点，则切换为 active 模式；
//运行中，active 节点开启心跳服务端，每 5s 广播 arp 地址；
//运行中，standby 节点，连接虚拟节点访问 active 服务器，如果连接成功，继续检测
//运行中，standby 节点，连接虚拟节点，如果连接失败，重试 3 次，则提升自己为 active 节点，执行 active 运行任务；
package floating_ip

import (
	"context"
	log "github.com/cihub/seelog"
	"github.com/j-keck/arping"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/net"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/service/heartbeat"
	"math/rand"
	net1 "net"
	"os/exec"
	"runtime"
	"time"
)

type FloatingIPConfig struct {
	Enabled         bool                 `config:"enabled"`
	IP              string               `config:"ip"`
	LocalIP              string           `config:"local_ip"`
	Netmask         string               `config:"netmask"`
	Interface       string               `config:"interface"`
	EchoPort        int                  `config:"echo_port"` //61111
	Priority        int                  `config:"priority"`
	BoradcastConfig config.NetworkConfig `config:"broadcast"`
}

type FloatingIPPlugin struct {
}

func (this FloatingIPPlugin) Name() string {
	return "floating_ip"
}

var (
	floatingIPConfig = FloatingIPConfig{
		Enabled:  false,
		Netmask:  "255.255.255.0",
		EchoPort: 61111,
		BoradcastConfig: config.NetworkConfig{
			Binding: "224.3.2.2:7654",
		},
	}
)

func (module FloatingIPPlugin) Setup(cfg *config.Config) {
	ok, err := env.ParseConfig("floating_ip", &floatingIPConfig)
	if ok && err != nil {
		panic(err)
	}

	if !floatingIPConfig.Enabled{
		return
	}

	if floatingIPConfig.Interface == "" || floatingIPConfig.IP == "" {
		//let's do some magic
		dev, ip, mask, err := util.GetPublishNetworkDeviceInfo()
		if err != nil {
			panic(err)
		}

		floatingIPConfig.LocalIP=ip

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

	if floatingIPConfig.IP == "" || floatingIPConfig.Interface == "" {
		panic("invalid floating_ip config")
	}

	if floatingIPConfig.Priority < 1 {
		floatingIPConfig.Priority = rand.Intn(1000)
	}
}

var pingTimeout = []string{"timeout", "Unreachable", "unreachable"}
var pingAlive = []string{"ttl", "time="}

func pingActiveNode(ip string) bool {
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ping", ip, "-i 1").Output()
	if err != nil {
		log.Debug(err, string(util.EscapeNewLine(out)))
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

//TODO handle two active nodes
//TODO support switch back to standby mode
func (module FloatingIPPlugin) SwitchToActiveMode() {

	log.Debugf("active floating_ip at: %v", floatingIPConfig.IP)

	err := net.SetupAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
	if err != nil {
		panic(err)
	}

	log.Tracef("floating_ip echo service :%v is up and running.", floatingIPConfig.EchoPort)

	//announce floating_ip, do arping every 10s
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
					log.Tracef("error to announce floating_ip,", v)
				}
			}
		}()

		for {
			select {
			case quit := <-arpSignal:
				if quit {
					log.Tracef("quit arping")
					return
				}
			default:
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
	}()

	//announce via broadcast
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
					log.Tracef("error to announce floating_ip,", v)
				}
			}
		}()

		req:=Request{
			IP: floatingIPConfig.LocalIP,
			Priority: floatingIPConfig.Priority,
		}

		for {
			select {
			case quit := <-multicastSignal:
				if quit {
					log.Tracef("quit broadcast")

					return
				}
			default:
				log.Trace("announce floating_ip, do broadcast every 10s")

				Broadcast(floatingIPConfig.BoradcastConfig,&req)

				time.Sleep(10 * time.Second)
			}
		}
	}()

	actived = true
	log.Infof("floating_ip listen at: %v , %v, %v", floatingIPConfig.IP, floatingIPConfig.EchoPort, floatingIPConfig.Priority)
}

func (module FloatingIPPlugin) Deactivate(silence bool) {
	if actived || silence {
		log.Debugf("deactivating floating_ip at: %v", floatingIPConfig.IP)
		err := net.DisableAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
		if err != nil && !silence {
			log.Error(err)
		}

		if actived{
			srvSignal <- true
			multicastSignal <- true
		}

		log.Tracef("floating_ip at: %v deactivated", floatingIPConfig.IP)
	}
	actived = false
}

func (module FloatingIPPlugin) SwitchToStandbyMode() {

	module.Deactivate(false)

	log.Infof("floating IP enter standby mode")

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
					log.Trace("error to switch to stand mode,", v)
				}
			}
		}()

		aliveChan := make(chan bool)
		go func() {
			heartbeat.StartClient(floatingIPConfig.IP, floatingIPConfig.EchoPort, func() {
				aliveChan <- true
			}, func() {
				aliveChan <- false
			})
		}()

	WAIT:

		alive := <-aliveChan
		if !alive {
			module.SwitchToActiveMode()
		} else {
			goto WAIT
		}
	}()

}

var actived bool

func (module FloatingIPPlugin) Start() error {

	if !floatingIPConfig.Enabled {
		log.Trace("floating ip disabled")
		return nil
	}

	log.Debugf("setup floating IP, root privilege are required")

	if !util.HasSudoPermission() {
		return errors.New("root privilege are required to use floating ip.")
	}

	//start heart server
	go func() {
		err := heartbeat.StartServer("0.0.0.0", floatingIPConfig.EchoPort)
		if err != nil {
			panic(err)
		}
	}()

	//start broadcast listener
	go ServeMulticastDiscovery(floatingIPConfig.BoradcastConfig, func(addr *net1.UDPAddr, n int, bytes []byte) {

		if !actived{
			log.Tracef("i am standby, no bother multicast message")
			return
		}

		//我是 master，别人也是 master
		//我不是 master，忽略
		//我是 master，忽略
		v := Request{}

		util.MustFromJSONBytes(bytes[:n], &v)

		log.Tracef("received multicast message: %v", util.ToJson(v, false))

		if v.IP==floatingIPConfig.LocalIP{
			log.Tracef("received my message: %v", util.ToJson(v, false))
			return
		}else{
			if v.Priority>=floatingIPConfig.Priority{
				log.Tracef("received high priority message, switch to backup mode: %v", util.ToJson(v, false))

				module.SwitchToStandbyMode()
			}
		}


	})

	//stop previous unclean status
	module.Deactivate(true)

	module.StateMachine()

	return nil
}

type State string

const Active State="Active"
const Backup State="Backup"
const Candidate State="Candidate"
const PreviousActiveIsBack State="PreviousActiveIsBack"

func (module FloatingIPPlugin) StateMachine() {

	//go func() {
	//
	//	var state State
	//	for {
	//		switch state {
	//		case Active:
	//			//
	//			break
	//		case Backup:
	//			break
	//		case Candidate:
	//			break
	//		case PreviousActiveIsBack:
	//			break
	//		}


			aliveChan := make(chan bool)
			go func() {
				err := heartbeat.StartClient(floatingIPConfig.IP, floatingIPConfig.EchoPort, func() {
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
				//target floating_ip can't connect, check ip address
				if pingActiveNode(floatingIPConfig.IP) {
					panic(errors.Errorf("the floating_ip [%v] has already been used by someone, but the gateway service is not running.", floatingIPConfig.IP))
				}
			}

			log.Tracef("active floating_ip node found: %v", alive)

			if alive {
				module.SwitchToStandbyMode()
			} else {
				module.SwitchToActiveMode()
			}

	//	}
	//}()
}

func (module FloatingIPPlugin) Stop() error {
	if !floatingIPConfig.Enabled {
		return nil
	}

	log.Tracef("stopping floating ip module")

	if actived {
		module.Deactivate(false)
	} else {
		haCheckSignal <- true
	}
	log.Tracef("floating ip module stopped")
	return nil
}
