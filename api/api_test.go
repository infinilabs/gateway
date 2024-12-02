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

package api

import (
	"fmt"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/common"
	"testing"
)

func TestDecodeEntryConfig(t *testing.T) {

	body := "{ \"id\": \"myid\", \"name\": \"my_es_entry\", \"enabled\": true, \"router\": \"c0oc4kkgq9s8qss2uk60\", \"max_concurrency\": 10, \"read_timeout\": 100, \"write_timeout\": 200, \"idle_timeout\": 300, \"read_buffer_size\": 1048576, \"write_buffer_size\": 1048576, \"tcp_keepalive\": true, \"tcp_keepalive_in_seconds\": 1, \"max_request_body_size\": 1048576, \"reduce_memory_usage\": false, \"network\": { \"binding\": \"127.0.0.1:9000\", \"host\": \"127.0.0.1\", \"port\": 9000, \"publish\": \"192.168.3.10:8000\", \"skip_occupied_port\": true, \"reuse_port\": true }, \"tls\": { \"enabled\": false } }"

	fmt.Println(body)
	cfg := common.EntryConfig{}
	err := util.FromJson(body, &cfg)
	fmt.Println(err)

	fmt.Println(cfg.ID)
	fmt.Println(cfg.Name)

}
