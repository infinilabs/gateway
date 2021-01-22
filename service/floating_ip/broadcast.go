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

package floating_ip

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/util"
	"net"
	log "src/github.com/cihub/seelog"
	"time"
)

const (
	maxDataSize = 4096
)

type Request struct {
	IP string  `json:"ip"`
	Priority int  `json:"priority"`
}

var lastBroadcast time.Time
//send a Broadcast message to network to discovery the cluster
func Broadcast(config config.NetworkConfig, req *Request) {
	if time.Now().Sub(lastBroadcast).Seconds() < 5 {
		log.Warn("broadcast requests was throttled(5s)")
		return
	}
	addr, err := net.ResolveUDPAddr("udp", config.GetBindingAddr())
	if err != nil {
		log.Error(err)
		return
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Error(err)
		return
	}

	payload := util.ToJSONBytes(req)

	_,err=c.Write(payload)
	if err != nil {
		log.Error(err)
		return
	}
	lastBroadcast=time.Now()
}

func ServeMulticastDiscovery(config config.NetworkConfig, h func(*net.UDPAddr, int, []byte)) {

	addr, err := net.ResolveUDPAddr("udp", config.GetBindingAddr())
	if err != nil {
		log.Error(err)
		return
	}

	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		log.Error(err)
		return
	}

	err=l.SetReadBuffer(maxDataSize)
	if err != nil {
		log.Error(err)
		return
	}

	for {
		b := make([]byte, maxDataSize)
		n, src, err := l.ReadFromUDP(b)
		if err != nil {
			log.Error("read from UDP failed:", err)
		}
		h(src, n, b)
	}

}

