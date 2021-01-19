//启动，如果是 active 模式，且没有存在虚拟节点，则切换为 standby 模式；
//启动，如果是 standby 模式，如果没有存在虚拟节点，则切换为 active 模式；
//运行中，active 节点开启心跳服务端，每 5s 广播 arp 地址；
//运行中，standby 节点，连接虚拟节点访问 active 服务器，如果连接成功，继续检测
//运行中，standby 节点，连接虚拟节点，如果连接失败，重试 3 次，则提升自己为 active 节点，执行 active 运行任务；
package floating_ip

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/j-keck/arping"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/net"
	"infini.sh/framework/core/util"
	net1 "net"
	"os/exec"
	"runtime"
	"time"
)

type FloatingIPConfig struct {
	Enabled   bool   `config:"enabled"`
	IP        string `config:"ip"`
	Netmask   string `config:"netmask"`
	Interface string `config:"interface"`
	EchoPort  string `config:"echo_port"` //61111
	Priority  int    `config:"priority"`
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
		EchoPort: "61111",
	}
)

func (module FloatingIPPlugin) Setup(cfg *config.Config) {
	ok, err := env.ParseConfig("floating_ip", &floatingIPConfig)
	if ok && err != nil {
		panic(err)
	}

	if floatingIPConfig.Interface==""||floatingIPConfig.IP==""{
		//let's do some magic
		dev,ip,mask,err:=util.GetPublishNetworkDeviceInfo()
		if err!=nil{
			panic(err)
		}

		if floatingIPConfig.Interface==""{
			floatingIPConfig.Interface=dev
		}

		log.Tracef("local publish address: %v,%v,%v",dev,ip,mask)

		//if mask is not setting, try guess
		if floatingIPConfig.Netmask==""{
			floatingIPConfig.Netmask=mask
		}

		if floatingIPConfig.IP==""{
			prefix:=util.GetIPPrefix(ip)
			floatingIPConfig.IP=prefix+".234"
		}

		log.Debugf("try to use floating ip address: %v,%v,%v",dev,floatingIPConfig.IP,mask)
	}

	if floatingIPConfig.IP==""||floatingIPConfig.Interface==""{
		panic("invalid floating_ip config")
	}

}

var pingTimeout=[]string{"timeout","Unreachable","unreachable"}

func pingActiveNode(ip string)bool  {
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	out, _ := exec.CommandContext(ctx,"ping", ip, "-i 3").Output()

	if util.ContainsAnyInArray(string(out), pingTimeout) {
		return false
	} else {
		return true
	}

}

func connectActiveNode() bool {
	conn, err := net1.DialTimeout("tcp", fmt.Sprintf("%s:%s", floatingIPConfig.IP, floatingIPConfig.EchoPort), time.Millisecond*200)
	if conn != nil {
		conn.Close()
	}
	if err != nil {
		return false
	}
	return true
}

func (module FloatingIPPlugin) IsActiveStillAlive() bool {
	//check three times on failure
	for i := 0; i < 3; i++ {
		if connectActiveNode() {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}

	return false
}

var srvSignal = make(chan bool,10)
var arpSignal = make(chan bool,10)
var haCheckSignal = make(chan bool,10)

//TODO handle two active nodes
//TODO support switch back to standby mode
func (module FloatingIPPlugin) SwitchToActiveMode() {

	log.Debugf("active floating_ip at: %v", floatingIPConfig.IP)

	err := net.SetupAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
	if err != nil {
		panic(err)
	}

	ln, err := net1.Listen("tcp", ":"+floatingIPConfig.EchoPort)
	if err != nil {
		panic(err)
	}

	log.Tracef("floating_ip echo service :%v is up and running.",floatingIPConfig.EchoPort)

	//start health check service
	go func() {
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {

					if r == nil {
						return
					}
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error to switch to active mode,", v)
				}
			}
		}()

		for {
			select {
			case quit := <-srvSignal:
				if quit {
					return
				}
			default:
				conn, _ := ln.Accept()
				if conn != nil {
					time.Sleep(200 * time.Millisecond)
					conn.Close()
				}
			}
		}
	}()

	//announce floating_ip, do arping every 10s
	go func() {
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {

					if r == nil {
						return
					}
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error to announce floating_ip,", v)
				}
			}
		}()

		for {
			select {
			case quit := <-arpSignal:
				if quit {
					return
				}
			default:
				log.Trace("announce floating_ip, do arping every 10s")
				ip := net1.ParseIP(floatingIPConfig.IP)
				err := arping.GratuitousArpOverIfaceByName(ip, floatingIPConfig.Interface)
				if err != nil {
					if util.ContainStr(err.Error(),"unable to open"){
						panic("please make sure running as root user, or sudo")
					}
					panic(err)
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()

	actived = true
	log.Infof("floating_ip listen at: %v, echo port: %v", floatingIPConfig.IP,floatingIPConfig.EchoPort)
}

func (module FloatingIPPlugin) Deactivate(silence bool) {
	if actived||silence{
		log.Debugf("deactivating floating_ip at: %v", floatingIPConfig.IP)
		err := net.DisableAlias(floatingIPConfig.Interface, floatingIPConfig.IP, floatingIPConfig.Netmask)
		if err != nil &&!silence {
			log.Error(err)
		}
		srvSignal <- true
		arpSignal <- true
		log.Tracef("floating_ip at: %v deactivated", floatingIPConfig.IP)
	}
	actived = false
}

func (module FloatingIPPlugin) SwitchToStandbyMode() {

	module.Deactivate(false)

	log.Debugf("floating IP enter standby mode")

	go func() {

		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {

					if r == nil {
						return
					}
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Error("error to switch to stand mode,", v)
				}
			}
		}()

		for {
			select {
			case quit := <-haCheckSignal:
				if quit {
					return
				}
			case <-time.After(time.Millisecond * 1000):
				if !module.IsActiveStillAlive() {
					log.Infof("floating_ip activated from standby mode")
					module.SwitchToActiveMode()
					return
				}
			}
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

	//stop previous unclean status
	module.Deactivate(true)

	//check active status
	alive := module.IsActiveStillAlive()

	if !alive{
		//target floating_ip can't connect, check ip address
		if pingActiveNode(floatingIPConfig.IP){
			panic(errors.Errorf("the floating_ip [%v] has already been used by someone, but the gateway service is not running.",floatingIPConfig.IP))
		}
	}

	log.Tracef("active floating_ip node found: %v", alive)

	if alive {
		module.SwitchToStandbyMode()
	} else {
		module.SwitchToActiveMode()
	}

	return nil
}


func (module FloatingIPPlugin) Stop() error {
	if !floatingIPConfig.Enabled {
		return nil
	}

	log.Tracef("stopping floating ip module")

	if actived {
		module.Deactivate(false)
	}else{
		haCheckSignal <- true
	}
	log.Tracef("floating ip module stopped")
	return nil
}
