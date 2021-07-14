package translog

import (
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/rotate"
)


type TranslogModule struct {
}


func (this TranslogModule) Name() string {
	return "translog"
}

func (module TranslogModule) Setup(cfg *Config) {

}

func (module TranslogModule) Start() error {

	return nil
}

func (module TranslogModule) Stop() error {

	rotate.Close()

	return nil
}
