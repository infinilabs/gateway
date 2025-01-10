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
	config3 "infini.sh/framework/core/config"
	"infini.sh/gateway/common"
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
