package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	xcap "infini.sh/gateway/captn"
	"infini.sh/gateway/translog"
	"os"
	capnp "zombiezen.com/go/capnproto2"
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
	scanner.Split(SplitFunc)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
			panic(err)
		}
		data := scanner.Bytes()
		msg, err := capnp.Unmarshal(data)
		if err != nil {
			panic(err)
		}
		requestGroup, err := xcap.ReadRootRequestGroup(msg)
		if err != nil {
			panic(err)
		}
		fmt.Println("request:")
		fmt.Println(requestGroup.Requests())
	}

}

func main() {
	flag.Parse()
	readRequests()
}
