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

package heartbeat

// golang achieve tcp long heartbeat connection with
// client
import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"net"
	"runtime"
	"time"
)

var (
	Req_REGISTER byte = 1 // 1 --- c register cid
	Res_REGISTER byte = 2 // 2 --- s response

	Req_HEARTBEAT byte = 3 // 3 --- s send heartbeat req
	Res_HEARTBEAT byte = 4 // 4 --- c send heartbeat res

	Req byte = 5 // 5 --- cs send data
	Res byte = 6 // 6 --- cs send ack
)

type Client struct {
	Dch chan bool
	Rch chan []byte
	Wch chan []byte
}

func New() *Client {
	client := Client{
		Dch: make(chan bool),
		Rch: make(chan []byte),
		Wch: make(chan []byte),
	}
	return &client
}

func (client *Client) Stop() {
	close(client.Dch)
	close(client.Rch)
	close(client.Wch)
}

func (client *Client) Start(host string, port int, dialTimeoutInMs, rwTimeoutInMs int, onConnect, onDisconnect func()) error {
	coo, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", host, port), time.Duration(dialTimeoutInMs)*time.Millisecond)
	if err != nil {
		onDisconnect()
		return err
	}

	conn, ok := coo.(*net.TCPConn)
	if !ok {
		onDisconnect()
		return err
	}

	onConnect()
	defer conn.Close()
	go client.ClientHandler(rwTimeoutInMs, conn)
	select {
	case <-client.Dch:
		onDisconnect()
	}
	return nil
}

func (client *Client) ClientHandler(rwTimeoutInMs int, conn *net.TCPConn) {
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
				log.Error("error on heartbeat client,", v)
			}
		}
	}()

	// Until register ok
	data := make([]byte, 128)
	for {
		conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_, err := conn.Write([]byte{Req_REGISTER, '#', '2'})
		if err != nil {
			client.Dch <- true
			break
		}

		conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_, err = conn.Read(data)
		if err != nil {
			client.Dch <- true
			break
		}
		if data[0] == Res_REGISTER {
			break
		}
	}
	go client.ClientRHandler(rwTimeoutInMs, conn)
	go client.ClientWHandler(rwTimeoutInMs, conn)
	go client.ClientWork()
}

func (client *Client) ClientRHandler(rwTimeoutInMs int, conn *net.TCPConn) {
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
				log.Error("error on heartbeat client,", v)
			}
		}
	}()

	for {
		// heartbeat packets, ack reply
		data := make([]byte, 128)
		err := conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		if err != nil {
			client.Dch <- true
			return
		}
		i, err := conn.Read(data)
		if err != nil {
			client.Dch <- true
			return
		}
		if i == 0 {
			client.Dch <- true
			return
		}
		err = conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		if err != nil {
			client.Dch <- true
			return
		}
		if data[0] == Req_HEARTBEAT {
			_, err = conn.Write([]byte{Res_REGISTER, '#', 'h'})
			if err != nil {
				client.Dch <- true
				return
			}
		} else if data[0] == Req {
			client.Rch <- data[2:]
			_, err = conn.Write([]byte{Res, '#'})
			if err != nil {
				client.Dch <- true
				return
			}
		}
	}
}

func (client *Client) ClientWHandler(rwTimeoutInMs int, conn net.Conn) {
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
				log.Error("error on heartbeat client,", v)
			}
		}
	}()

	for {
		select {
		case msg := <-client.Wch:
			err := conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
			if err != nil {
				client.Dch <- true
				return
			}
			_, err = conn.Write(msg)
			if err != nil {
				client.Dch <- true
				return
			}
		}
	}

}

func (client *Client) ClientWork() {
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
				log.Error("error on heartbeat client,", v)
			}
		}
	}()

	for {
		select {
		case _ = <-client.Rch:
			client.Wch <- []byte{Req, '#', 'x', 'x', 'x', 'x', 'x'}
		}
	}
}
