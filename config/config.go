package config

type ProxyConfig struct {
	Enabled       bool   `config:"enabled"`
	Elasticsearch string `config:"elasticsearch"`
	Balancer     string   `config:"balancer"`
	Timeout      string   `config:"timeout"`
	MaxConnection       int `config:"max_connection"`
	MaxResponseBodySize int `config:"max_response_size"`

	Weights  map[string]int `config:"weight"`
	Discover struct {
		Enabled    bool     `config:"enabled"`
		NodeFilter []string `config:"node_filter"`
	} `config:"discovery"`

}

type CacheConfig struct {
	Enabled        bool   `config:"enabled"`
	Type           string `config:"type"` //redis,local
	TTL            string `config:"ttl"`
	AsyncSearchTTL string `config:"async_search_ttl"`
	MaxCachedItem  int64  `config:"max_cached_item"`
}

