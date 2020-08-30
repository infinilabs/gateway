package main

import (
	"fmt"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/module"
	"os"
	"path"
	"path/filepath"
	"plugin"
)

func Discovery(dir string) {

	//load plugins
	pattern := path.Join(dir, "/*.so")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("plugins to load: ", matches)

	for _, v := range matches {
		fmt.Println("loading: ", v)
		plug, err := plugin.Open(v)
		if err != nil {
			fmt.Println(err)
			continue
		}

		dll, err := plug.Lookup("Plugin")
		if err != nil {
			fmt.Println(err)
			continue
		}

		plugin, ok := dll.(module.Module)
		if !ok {
			fmt.Println("unexpected type from module symbol")
			continue
		}

		module.RegisterUserPlugin(plugin)
	}
}

func main() {
	Discovery("../plugin")

}

func main1() {

	//load plugins
	pattern := "../plugin/*.so"

	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(matches)

	for _, v := range matches {
		fmt.Println("loading: ", v)
		plug, err := plugin.Open(v)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dll, err := plug.Lookup("Plugin")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var plugin module.Module
		plugin, ok := dll.(module.Module)
		if !ok {
			fmt.Println("unexpected type from module symbol")
			os.Exit(1)
		}

		fmt.Println(plugin.Name())

		cfg := env.GetModuleConfig(plugin.Name())

		fmt.Println(cfg)

		plugin.Setup(cfg)
		err = plugin.Start()
		if err != nil {
			panic(err)
		}
	}

}
