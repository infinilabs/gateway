package elastic

import "time"

type ProxyConfig struct {
	Enabled             bool   `config:"enabled"`
	Elasticsearch       string `config:"elasticsearch"`
	Balancer            string `config:"balancer"`
	Timeout             string `config:"timeout"`
	MaxConnection       int    `config:"max_connection"`
	MaxResponseBodySize int    `config:"max_response_size"`

	MaxConnWaitTimeout  time.Duration `config:"max_conn_wait_timeout"`
	MaxIdleConnDuration time.Duration `config:"max_idle_conn_duration"`
	MaxConnDuration     time.Duration `config:"max_conn_duration"`
	ReadTimeout         time.Duration `config:"read_timeout"`
	WriteTimeout        time.Duration `config:"write_timeout"`

	ReadBufferSize        int  `config:"read_buffer_size"`
	WriteBufferSize       int  `config:"write_buffer_size"`
	TLSInsecureSkipVerify bool `config:"tls_insecure_skip_verify"`

	Weights  map[string]int `config:"weight"`
	Discover struct {
		Enabled bool `config:"enabled"`
		Refresh struct {
			Enabled  bool   `config:"enabled"`
			Interval string `config:"interval"`
		} `config:"refresh"`
		Tags struct {
			Exclude []string `config:"exclude"`
			Include []string `config:"include"`
		} `config:"tags"`
		Roles struct {
			Exclude []string `config:"exclude"`
			Include []string `config:"include"`
		} `config:"roles"`
	} `config:"discovery"`
}
