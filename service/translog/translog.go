package translog

import (
	"infini.sh/framework/core/rotate"
)


type TranslogModule struct {
}


func (this TranslogModule) Name() string {
	return "translog"
}

func (module TranslogModule) Setup() {

}

func (module TranslogModule) Start() error {

	//TODO
	// 生命周期管理
	// 定期上传到远程 s3 仓库

	return nil
}

func (module TranslogModule) Stop() error {

	rotate.Close()

	return nil
}
