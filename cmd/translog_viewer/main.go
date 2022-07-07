package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/proxy/output/translog"
	"os"
	"time"
)

var (
	file = flag.String("file", "/tmp/translog.bin", "the translog file to view")
)
var splitBytes = []byte(translog.SplitLine)
var searchLen = len(splitBytes)

func SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	dataLen := len(data)

	// Return nothing if at end of file and no data passed
	if atEOF && dataLen == 0 {
		return 0, nil, nil
	}

	// Find next separator and return token
	if i := bytes.Index(data, splitBytes); i >= 0 {
		return i + searchLen, data[0:i], nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return dataLen, data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func readRequests() {

	file, err := os.Open(*file)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 20*1024*1024), 20*1024*1024)
	scanner.Split(SplitFunc)

	fmt.Println("read files:",file)

	for scanner.Scan() {

		fmt.Println("scanning")

		if err := scanner.Err(); err != nil {
			fmt.Println(err)
			panic(err)
		}
		data := scanner.Bytes()
		req:=fasthttp.AcquireRequest()
		err:=req.Decode(data)
		if err != nil {
			panic(err)
		}
		//msg, err := capnp.Unmarshal(data)
		//if err != nil {
		//	panic(err)
		//}
		//requestGroup, err := xcap.ReadRootRequestGroup(msg)
		//if err != nil {
		//	panic(err)
		//}
		fmt.Println("request:")
		fmt.Println(req.String())
		//fmt.Println(requestGroup.Requests())
	}

	fmt.Println("finished.")
}

func main() {
	flag.Parse()
	readRequests()
	time.Sleep(5*time.Second)
}
