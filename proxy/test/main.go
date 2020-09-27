package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/dgraph-io/ristretto"
	"time"
)
//
//func Discovery(dir string) {
//
//	//load plugins
//	pattern := path.Join(dir, "/*.so")
//	matches, err := filepath.Glob(pattern)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println("plugins to load: ", matches)
//
//	for _, v := range matches {
//		fmt.Println("loading: ", v)
//		plug, err := plugin.Open(v)
//		if err != nil {
//			fmt.Println(err)
//			continue
//		}
//
//		dll, err := plug.Lookup("Plugin")
//		if err != nil {
//			fmt.Println(err)
//			continue
//		}
//
//		plugin, ok := dll.(module.Module)
//		if !ok {
//			fmt.Println("unexpected type from module symbol")
//			continue
//		}
//
//		module.RegisterUserPlugin(plugin)
//	}
//}
//
//func main2() {
//	Discovery("../plugin")
//
//}
//
//func main1() {
//
//	//load plugins
//	pattern := "../plugin/*.so"
//
//	matches, err := filepath.Glob(pattern)
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	fmt.Println(matches)
//
//	for _, v := range matches {
//		fmt.Println("loading: ", v)
//		plug, err := plugin.Open(v)
//		if err != nil {
//			fmt.Println(err)
//			os.Exit(1)
//		}
//
//		dll, err := plug.Lookup("Plugin")
//		if err != nil {
//			fmt.Println(err)
//			os.Exit(1)
//		}
//
//		var plugin module.Module
//		plugin, ok := dll.(module.Module)
//		if !ok {
//			fmt.Println("unexpected type from module symbol")
//			os.Exit(1)
//		}
//
//		fmt.Println(plugin.Name())
//
//		cfg := env.GetModuleConfig(plugin.Name())
//
//		fmt.Println(cfg)
//
//		plugin.Setup(cfg)
//		err = plugin.Start()
//		if err != nil {
//			panic(err)
//		}
//	}
//
//}
//

func main()  {

	header:=[]byte("hello header")
	body:=[]byte("hello body")

	fmt.Println(header)
	fmt.Println(body)

	buffer:=bytes.Buffer{}

	headerLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerLength, uint32(len(header)))
	fmt.Println(headerLength)

	bodyLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(bodyLength, uint32(len(body)))
	fmt.Println(bodyLength)


	//header length
	buffer.Write(headerLength)
	buffer.Write(header)

	//body
	buffer.Write(bodyLength)
	buffer.Write(body)

	data:=buffer.Bytes()
	fmt.Println(data)
	buffer.Reset()

	readerHeaderLengthBytes := make([]byte, 4)
	reader:=bytes.NewBuffer(data)
	n,err:=reader.Read(readerHeaderLengthBytes)
	if err!=nil{
		fmt.Println(n,err)
	}

	fmt.Println(readerHeaderLengthBytes)

	readerHeaderLength:=binary.LittleEndian.Uint32(readerHeaderLengthBytes)
	readerHeader := make([]byte,readerHeaderLength )
	n,err=reader.Read(readerHeader)
	if err!=nil{
		fmt.Println(n,err)
	}

	fmt.Println(readerHeader)



	readerBodyLengthBytes := make([]byte, 4)
	n,err=reader.Read(readerBodyLengthBytes)
	if err!=nil{
		fmt.Println(n,err)
	}

	fmt.Println(readerBodyLengthBytes)

	readerBodyLength:=binary.LittleEndian.Uint32(readerBodyLengthBytes)
	readerBody := make([]byte,readerBodyLength )
	n,err=reader.Read(readerBody)
	if err!=nil{
		fmt.Println(n,err)
	}

	fmt.Println(readerBody)



}

func main1() {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Num keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}

	cache.Set("key", "value", 1) // set a value
	// wait for value to pass through buffers
	time.Sleep(10 * time.Millisecond)

	value, found := cache.Get("key")
	if !found {
		panic("missing value")
	}
	fmt.Println(value)
	cache.Del("key")
}
