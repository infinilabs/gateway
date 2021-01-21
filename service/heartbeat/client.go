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

var rwTimeoutInMs = 10000
var dialTimeoutInMs = 1000

func StartClient(host string, port int, onConnect, onDisconnect func()) error {

	Dch = make(chan bool)
	Rch = make(chan []byte)
	Wch = make(chan []byte)
	//addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%v",host,port))
	//log.Error("dial tcp",err)

	//conn, err := net.DialTCP("tcp", nil, addr)
	coo, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", host, port), time.Duration(dialTimeoutInMs)*time.Millisecond)
	if err != nil {
		onDisconnect()
	}

	conn, ok := coo.(*net.TCPConn)
	if !ok {
		//log.Error("not ok")
		onDisconnect()
		return err
	}

	//log.Error("dial tcp",err)
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
	go ClientHandler(conn)
	select {
	case <-Dch:
		//fmt.Println("close connection")
		onDisconnect()

		//TODO remote closed
	}
	return nil
}

func ClientHandler(conn *net.TCPConn) {
	// Until register ok
	data := make([]byte, 128)
	for {
		conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_,err:=conn.Write([]byte{Req_REGISTER, '#', '2'})
		if err!=nil{
			//log.Error(err)
			Dch <- true
			break
		}

		conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		_,err=conn.Read(data)
		if err!=nil{
			//log.Error(err)
			Dch <- true
			break
		}
				//fmt.Println(string(data))
		if data[0] == Res_REGISTER {
			break
		}
	}
		//fmt.Println("i'm register")
	go ClientRHandler(conn)
	go ClientWHandler(conn)
	go ClientWork()
}

func ClientRHandler(conn *net.TCPConn) {

	for {
		// heartbeat packets, ack reply
		data := make([]byte, 128)
		conn.SetReadDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		i, err := conn.Read(data)
		if err!=nil{
			//log.Error(err)
			Dch <- true
			return
		}
		if i == 0 {
			Dch <- true
			return
		}
		conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
		if data[0] == Req_HEARTBEAT {
			//fmt.Println("recv ht pack")
			_,err=conn.Write([]byte{Res_REGISTER, '#', 'h'})
			if err!=nil{
				//log.Error(err)
				Dch <- true
				return
			}
			//fmt.Println("send ht pack ack")
		} else if data[0] == Req {
			//fmt.Println("recv data pack")
			//fmt.Printf("%v\n", string(data[2:]))
			Rch <- data[2:]
			_,err=conn.Write([]byte{Res, '#'})
			if err!=nil{
				//log.Error(err)
				Dch <- true
				return
			}
		}
	}
}

func ClientWHandler(conn net.Conn) {
	for {
		select {
		case msg := <-Wch:
			//fmt.Println((msg[0]))
			//fmt.Println("send data after: " + string(msg[1:]))
			conn.SetWriteDeadline(time.Now().Add(time.Duration(rwTimeoutInMs) * time.Millisecond))
			_,err:=conn.Write(msg)
			if err!=nil{
				//log.Error(err)
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
			//fmt.Println("work recv " + string(msg))
			//fmt.Println("work recv ___")

			Wch <- []byte{Req, '#', 'x', 'x', 'x', 'x', 'x'}
		}
	}
}
