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
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"net"
	"time"
)

const (
	maxDataSize = 4096
)

type Request struct {
	IsActive bool 	  `json:"active"`
	FloatingIP string `json:"floating_ip"`
	FixedIP    string `json:"fixed_ip"`
	EchoPort   int    `json:"echo_port"`
	Priority   int    `json:"priority"`
}

var lastBroadcast time.Time
//send a Broadcast message to network to discovery the cluster
func Broadcast(config *FloatingIPConfig, req *Request) {
	if config==nil{
		panic("invalid config")
	}

	if time.Now().Sub(lastBroadcast).Seconds() < 1 {
		log.Warn("broadcast requests was throttled(5s)")
		return
	}
	addr, err := net.ResolveUDPAddr("udp", config.BroadcastConfig.GetBindingAddr())
	if err != nil {
		log.Error(err)
		return
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Error(err)
		return
	}

	payload := util.MustToJSONBytes(req)

	_,err=c.Write(payload)
	if err != nil {
		log.Error(err)
		return
	}
	lastBroadcast=time.Now()
}

func ServeMulticastDiscovery(config *FloatingIPConfig, h func(*net.UDPAddr, int, []byte)) {

	if config==nil{
		panic("invalid config")
	}

	addr, err := net.ResolveUDPAddr("udp", config.BroadcastConfig.GetBindingAddr())
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

