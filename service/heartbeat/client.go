package heartbeat

// golang achieve tcp long heartbeat connection with
// client
import (
	"fmt"
	"net"
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

var Dch chan bool
var Rch chan []byte
var Wch chan []byte

func StartClient(host string, port int,dialTimeoutInMs,rwTimeoutInMs int, onConnect, onDisconnect func()) error {

	Dch = make(chan bool)
	Rch = make(chan []byte)
	Wch = make(chan []byte)
	coo, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", host, port), time.Duration(dialTimeoutInMs)*time.Millisecond)
	if err != nil {
		onDisconnect()
	}

	conn, ok := coo.(*net.TCPConn)
	if !ok {
		onDisconnect()
		return err
	}

	if err != nil {
		onDisconnect()
		return err
	}

	onConnect()
	defer conn.Close()
	go ClientHandler(rwTimeoutInMs,conn)
	select {
	case <-Dch:
		onDisconnect()
	}
	return nil
}

func ClientHandler(rwTimeoutInMs int,conn *net.TCPConn) {
	// Until register ok
	data := make([]byte, 128)
	for {
		conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_,err:=conn.Write([]byte{Req_REGISTER, '#', '2'})
		if err!=nil{
			Dch <- true
			break
		}

		conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_,err=conn.Read(data)
		if err!=nil{
			Dch <- true
			break
		}
		if data[0] == Res_REGISTER {
			break
		}
	}
	go ClientRHandler(rwTimeoutInMs,conn)
	go ClientWHandler(rwTimeoutInMs,conn)
	go ClientWork()
}

func ClientRHandler(rwTimeoutInMs int,conn *net.TCPConn) {

	for {
		// heartbeat packets, ack reply
		data := make([]byte, 128)
		conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		i, err := conn.Read(data)
		if err!=nil{
			Dch <- true
			return
		}
		if i == 0 {
			Dch <- true
			return
		}
		conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		if data[0] == Req_HEARTBEAT {
			_,err=conn.Write([]byte{Res_REGISTER, '#', 'h'})
			if err!=nil{
				Dch <- true
				return
			}
		} else if data[0] == Req {
			Rch <- data[2:]
			_,err=conn.Write([]byte{Res, '#'})
			if err!=nil{
				Dch <- true
				return
			}
		}
	}
}

func ClientWHandler(rwTimeoutInMs int,conn net.Conn) {
	for {
		select {
		case msg := <-Wch:
			conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
			_,err:=conn.Write(msg)
			if err!=nil{
				Dch <- true
				return
			}
		}
	}

}

func ClientWork() {
	for {
		select {
		case _ = <-Rch:
			Wch <- []byte{Req, '#', 'x', 'x', 'x', 'x', 'x'}
		}
	}
}
