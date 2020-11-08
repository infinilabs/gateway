package entry

import (
	config3 "infini.sh/framework/core/config"
	config2 "infini.sh/gateway/config"
	"testing"
	"time"
)

func TestMulti(t *testing.T) {
	config := config2.EntryConfig{Enabled: true}
	config.Name = "test"
	config.MaxConcurrency = 100
	config.NetworkConfig = config3.NetworkConfig{Host: "0.0.0.0", Port: "8081"}

	entry := Entrypoint{
		config: config,
	}

	err := entry.Start()
	if err != nil {
		panic(err)
	}

	time.Sleep(5*time.Second)

	err= entry.Stop()
	if err != nil {
		panic(err)
	}
	time.Sleep(5*time.Second)
}