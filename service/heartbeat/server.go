package heartbeat

// golang achieve tcp long heartbeat connection with
// server
import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"net"
	"runtime"
	"sync"
	"time"
)

var (
//Req_REGISTER byte = 1 // 1 --- c register cid
//Res_REGISTER byte = 2 // 2 --- s response
//
//Req_HEARTBEAT byte = 3 // 3 --- s send heartbeat req
//Res_HEARTBEAT byte = 4 // 4 --- c send heartbeat res
//
//Req byte = 5 // 5 --- cs send data
//Res byte = 6 // 6 --- cs send ack
)

type CS struct {
	Rch chan []byte
	Wch chan []byte
	Dch chan bool
	u   string
}

func NewCs(uid string) *CS {
	return &CS{Rch: make(chan []byte), Wch: make(chan []byte), u: uid}
}

var CMap map[string]*CS

var lock sync.RWMutex

func StartServer(host string, port int) error {
	CMap = make(map[string]*CS)
	listen, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(host), port, ""})
	if err != nil {
		//fmt.Println("Listen Port failed:", err.Error())
		//TODO
		return err
	}
	//fmt.Println("initialized connection, connection requests from clients ...")
	go PushGRT()
	server(listen)
	return nil
}

func PushGRT() {
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

	for {
		time.Sleep(3 * time.Second)
		for _, v := range CMap {
			//fmt.Println("push msg to user:" + k)
			v.Wch <- []byte{Req, '#', 'p', 'u', 's', 'h', '!'}
		}
	}
}

func server(listen *net.TCPListener) {
	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			//fmt.Println("Accept client connection exception:", err.Error())
			continue
		}
		//fmt.Println("client connections from:", conn.RemoteAddr())
		// handler goroutine
		go ServerHandler(conn)
	}
}

func ServerHandler(conn net.Conn) {
	defer conn.Close()
	data := make([]byte, 128)
	var uid string
	var C *CS
	for {
		conn.Read(data)
		//fmt.Println("data sent from the client:", string(data))
		if data[0] == Req_REGISTER { // register
			conn.Write([]byte{Res_REGISTER, '#', 'o', 'k'})
			uid = string(data[2:])
			C = NewCs(uid)
			lock.Lock()
			CMap[uid] = C
			lock.Unlock()
			//fmt.Println("register client")
			//fmt.Println(uid)
			break
		} else {
			conn.Write([]byte{Res_REGISTER, '#', 'e', 'r'})
		}
	}
	//	ClientWHandler
	go ServerWHandler(conn, C)

	//	ClientRHandler
	go ServerRHandler(conn, C)

	//	Worker
	go ServerWork(C)
	select {
	case <-C.Dch:
		//fmt.Println("close handler goroutine")
	}
}

// write the data correctly
// timing detection conn die => goroutine die
func ServerWHandler(conn net.Conn, C *CS) {
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

	// read data written Wch of business ClientWork
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case d := <-C.Wch:
			conn.Write(d)
		case <-ticker.C:
			if _, ok := CMap[C.u]; !ok {
				//fmt.Println("conn die, close ClientWHandler")
				return
			}
		}
	}
}

// read client data heartbeat +
func ServerRHandler(conn net.Conn, C *CS) {
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

	// heartbeat ack
	// business data is written Wch

	for {
		data := make([]byte, 128)
		// setReadTimeout
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			//fmt.Println(err)
		}
		if _, derr := conn.Read(data); derr == nil {
			// message from the client might be confirmed
			// data messages
			//fmt.Println(data)
			if data[0] == Res {
				//fmt.Println("recv client data ack")
			} else if data[0] == Req {
				//fmt.Println("recv client data")
				//fmt.Println(data)
				conn.Write([]byte{Res, '#'})
				// C.Rch <- data
			}

			continue
		}

		conn.Write([]byte{Req_HEARTBEAT, '#'})
		//fmt.Println("send ht packet")
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, herr := conn.Read(data); herr == nil {
			// fmt.Println(string(data))
			//fmt.Println("resv ht packet ack")
		} else {
			lock.Lock()
			delete(CMap, C.u)
			lock.Unlock()
			//fmt.Println("delete user!")
			return
		}
	}
}

func ServerWork(C *CS) {
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

	time.Sleep(1 * time.Second)
	C.Wch <- []byte{Req, '#', 'h', 'e', 'l', 'l', 'o'}

	time.Sleep(1 * time.Second)
	C.Wch <- []byte{Req, '#', 'h', 'e', 'l', 'l', 'o'}
	// read information from the read ch
	/*	ticker := time.NewTicker(20 * time.Second)
				for {
					select {
					case d := <-C.Rch:
						C.Wch <- d
					case <-ticker.C:
						if _, ok := CMap[C.u]; !ok {
							return
						}
					}
				}
			 * /
		// write information to write ch
	*/
}
