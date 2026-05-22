// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package entry

import (
	"fmt"
	config3 "infini.sh/framework/core/config"
	"infini.sh/gateway/common"
	"net"
	"testing"
)

func TestMulti(t *testing.T) {
	config := common.EntryConfig{Enabled: true}
	config.Name = "test"
	config.MaxConcurrency = 100
	config.NetworkConfig = config3.NetworkConfig{Host: "127.0.0.1", Port: 8081}

	entry := Entrypoint{
		config: config,
	}

	err := entry.Start()
	if err != nil {
		panic(err)
	}

	err = entry.Stop()
	if err != nil {
		panic(err)
	}
}

func TestStartDoesNotHoldPortWhenFlowResolutionFails(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()

	routerName := "missing-flow-router"
	common.RegisterRouterConfig(common.RouterConfig{
		Name: routerName,
		Rules: []common.RuleConfig{
			{
				Method:      []string{"GET"},
				PathPattern: []string{"/"},
				Flow:        []string{"missing-flow"},
			},
		},
	})

	entry := Entrypoint{
		config: common.EntryConfig{
			Enabled:          true,
			Name:             "test-missing-flow",
			RouterConfigName: routerName,
			NetworkConfig: config3.NetworkConfig{
				Binding: fmt.Sprintf("127.0.0.1:%d", addr.Port),
			},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected start to panic when referenced flow is missing")
		}

		reuse, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", addr.Port))
		if err != nil {
			t.Fatalf("expected port to remain free after failed start: %v", err)
		}
		_ = reuse.Close()
	}()

	_ = entry.Start()
}
