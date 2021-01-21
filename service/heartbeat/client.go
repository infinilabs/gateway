package main

// golang achieve tcp long heartbeat connection with
// client
import (
	"fmt"
	"net"
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

func main() {
	Start("127.0.0.1",6666,func() {
		println("connected")
	}, func() {
		fmt.Println("disconnected")
	})
}

func Start(host string,port int,onConnect,onDisconnect func())error  {
	Dch = make(chan bool)
	Rch = make(chan []byte)
	Wch = make(chan []byte)
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%v",host,port))
	conn, err := net.DialTCP("tcp", nil, addr)
	//	conn, err := net.Dial("tcp", "127.0.0.1:6666")
	if err != nil {
		//fmt.Println("server connection failed:", err.Error())
		//TODO server is down
		onDisconnect()
		return err
	}

	//fmt.Println("connected servers")
	onConnect()
	//TODO connected

	defer conn.Close()
	go Handler(conn)
	select {
	case <-Dch:
		//fmt.Println("close connection")
		onDisconnect()

		//TODO remote closed
	}
	return nil
}


func Handler(conn *net.TCPConn) {
	// Until register ok
	data := make([]byte, 128)
	for {
		conn.Write([]byte{Req_REGISTER, '#', '2'})
		conn.Read(data)
		//		fmt.Println(string(data))
		if data[0] == Res_REGISTER {
			break
		}
	}
	//	fmt.Println("i'm register")
	go RHandler(conn)
	go WHandler(conn)
	go Work()
}

func RHandler(conn *net.TCPConn) {

	for {
		// heartbeat packets, ack reply
		data := make([]byte, 128)
		i, _ := conn.Read(data)
		if i == 0 {
			Dch <- true
			return
		}
		if data[0] == Req_HEARTBEAT {
			//fmt.Println("recv ht pack")
			conn.Write([]byte{Res_REGISTER, '#', 'h'})
			//fmt.Println("send ht pack ack")
		} else if data[0] == Req {
			//fmt.Println("recv data pack")
			//fmt.Printf("%v\n", string(data[2:]))
			Rch <- data[2:]
			conn.Write([]byte{Res, '#'})
		}
	}
}

func WHandler(conn net.Conn) {
	for {
		select {
		case msg := <-Wch:
			//fmt.Println((msg[0]))
			//fmt.Println("send data after: " + string(msg[1:]))
			conn.Write(msg)
		}
	}

}

func Work() {
	for {
		select {
		case _ = <-Rch:
			//fmt.Println("work recv " + string(msg))
			//fmt.Println("work recv ___")

			Wch <- []byte{Req, '#', 'x', 'x', 'x', 'x', 'x'}
		}
	}
}
