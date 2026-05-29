/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package managed

import "infini.sh/framework/core/module"

const moduleName = "managed_gateway_bootstrap"

func init() {
	module.RegisterModuleWithPriority(&ManagedModule{}, 100)
}

type ManagedModule struct{}

func (m *ManagedModule) Name() string {
	return moduleName
}

func (m *ManagedModule) Setup() {
	RegisterTokenExchangeCallback()
}

func (m *ManagedModule) Start() error {
	return nil
}

func (m *ManagedModule) Stop() error {
	return nil
}
