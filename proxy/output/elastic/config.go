package elastic

import "time"

type ProxyConfig struct {
	Elasticsearch string `config:"elasticsearch"`
	Balancer      string `config:"balancer"`

	MaxConnection         int  `config:"max_connection_per_node"`
	MaxResponseBodySize   int  `config:"max_response_size"`
	MaxRetryTimes         int  `config:"max_retry_times"`
	RetryOnBackendFailure bool `config:"retry_on_backend_failure"`
	RetryOnBackendBusy    bool `config:"retry_on_backend_busy"`
	RetryDelayInMs        int  `config:"retry_delay_in_ms"`

	MaxConnWaitTimeout    time.Duration `config:"max_conn_wait_timeout"`
	MaxIdleConnDuration   time.Duration `config:"max_idle_conn_duration"`
	MaxConnDuration       time.Duration `config:"max_conn_duration"`
	Timeout               time.Duration `config:"timeout"`
	DialTimeout           time.Duration `config:"dial_timeout"`
	ReadTimeout           time.Duration `config:"read_timeout"`
	WriteTimeout          time.Duration `config:"write_timeout"`
	ReadBufferSize        int           `config:"read_buffer_size"`
	WriteBufferSize       int           `config:"write_buffer_size"`
	TLSInsecureSkipVerify bool          `config:"tls_insecure_skip_verify"`

	FixedClient bool   `config:"fixed_client"`
	ClientMode  string `config:"client_mode"`

	SkipAvailableCheck                 bool `config:"skip_available_check"`
	CheckClusterHealthWhenNotAvailable bool `config:"check_cluster_health_when_not_available"`

	SkipKeepOriginalURI   bool `config:"skip_keep_original_uri"`
	SkipCleanupHopHeaders bool `config:"skip_cleanup_hop_headers"`
	SkipEnrichMetadata    bool `config:"skip_metadata_enrich"`

	Weights map[string]int `config:"weights"`

	Refresh struct {
		Enabled  bool   `config:"enabled"`
		Interval string `config:"interval"`
	} `config:"refresh"`

	Filter struct {
		Hosts struct {
			Exclude []string `config:"exclude"`
			Include []string `config:"include"`
		} `config:"hosts"`
		Tags struct {
			Exclude []map[string]interface{} `config:"exclude"`
			Include []map[string]interface{} `config:"include"`
		} `config:"tags"`
		Roles struct {
			Exclude []string `config:"exclude"`
			Include []string `config:"include"`
		} `config:"roles"`
	} `config:"filter"`
}
