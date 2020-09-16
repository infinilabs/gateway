package main

import (
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


func main() {
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
