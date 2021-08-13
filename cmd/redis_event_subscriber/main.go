package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"infini.sh/framework/core/util"
	"io"
	"runtime"
)

var host = flag.String("host", "127.0.0.1", "redis host")
var port = flag.Int("port", 6379, "redis port")
var channel = flag.String("channel", "gateway", "channel name")
var password = flag.String("password", "", "password")
var db = flag.Int("db", 0, "db")
var client *redis.Client

func main() {
	runtime.GOMAXPROCS(1)
	flag.Parse()

	fmt.Printf("subscribe to redis %v:%v channel:%v \n", *host, *port, *channel)

	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%v", *host, *port),
		Password: *password,
		DB:       *db,
	})

	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}

	for ;;{
		msg:=<-client.Subscribe(*channel).Channel()
		data:=[]byte(msg.Payload)
		err:=decodeRequest(data)
		if err!=nil{
			panic(err)
		}
	}

}

func readBytes(reader io.Reader,length uint32)[]byte  {
	bytes := make([]byte, length)
	_, err := reader.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}

func readBytesLength(reader io.Reader)uint32  {
	lengthBytes := make([]byte, 4)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(lengthBytes)
}

var colon = []byte(": ")
var newLine = []byte("\n")

func decodeRequest(data []byte)error {
	reader:=&bytes.Reader{}
	reader.Reset(data)

	////request
	fmt.Println("request:")

	//schema
	schemaLength:=readBytesLength(reader)
	schema:=readBytes(reader,schemaLength)

	fmt.Println("schema:",string(schema))

	//method
	methodLengthBytes := readBytesLength(reader)
	method:=readBytes(reader,methodLengthBytes)

	fmt.Println("method:",string(method))

	//uri
	uriLengthBytes := readBytesLength(reader)
	readerUri:=readBytes(reader,uriLengthBytes)

	fmt.Println("uri:",string(readerUri))


	//headers
	readerHeaderLengthBytes := readBytesLength(reader)
	readerHeader:=readBytes(reader,readerHeaderLengthBytes)

	fmt.Println("header:\n",string(readerHeader))

	//line := bytes.Split(readerHeader, newLine)
	//for _, l := range line {
	//	kv := bytes.Split(l, colon)
	//	if len(kv) == 2 {
	//		req.Header.SetBytesKV(kv[0], kv[1])
	//	}
	//}

	//body
	readerBodyLengthBytes := make([]byte, 4)
	_, err := reader.Read(readerBodyLengthBytes)
	if err != nil {
		return err
	}
	readerBodyLength := binary.LittleEndian.Uint32(readerBodyLengthBytes)
	if readerBodyLength>0{
		readerBody := make([]byte, readerBodyLength)
		_, err = reader.Read(readerBody)
		if err != nil {
			return err
		}
		fmt.Println("body:\n",string(readerBody))
	}


	if reader.Len()==0{
		return nil
	}

	////response
	fmt.Println("response:")

	responseReaderHeaderLengthBytes := make([]byte, 4)
	_, err = reader.Read(responseReaderHeaderLengthBytes)
	if err != nil {
		return err
	}

	readerHeaderLength := binary.LittleEndian.Uint32(responseReaderHeaderLengthBytes)
	readerHeader = make([]byte, readerHeaderLength)
	_, err = reader.Read(readerHeader)
	if err != nil {
		return err
	}
	fmt.Println("header:\n",string(readerHeader))

	//line := bytes.Split(readerHeader, newLine)
	//for _, l := range line {
	//	kv := bytes.Split(l, colon)
	//	if len(kv) == 2 {
	//		res.Header.SetBytesKV(kv[0], kv[1])
	//	}
	//}

	readerBodyLengthBytes = make([]byte, 4)
	_, err = reader.Read(readerBodyLengthBytes)
	if err != nil {
		return err
	}

	readerBodyLength = binary.LittleEndian.Uint32(readerBodyLengthBytes)
	if readerBodyLength>0{
		readerBody := make([]byte, readerBodyLength)
		_, err = reader.Read(readerBody)
		if err != nil {
			return err
		}
		fmt.Println("body:\n",string(readerBody))
	}

	statusCode := make([]byte, 4)
	_, err = reader.Read(statusCode)
	if err != nil {
		return err
	}

	fmt.Println("status:",int(util.BytesToUint32(statusCode)))

	return nil
}
