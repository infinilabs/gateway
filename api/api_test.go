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
