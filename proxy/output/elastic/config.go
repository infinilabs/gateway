package elastic

type ProxyConfig struct {
	Enabled             bool   `config:"enabled"`
	Elasticsearch       string `config:"elasticsearch"`
	Balancer            string `config:"balancer"`
	Timeout             string `config:"timeout"`
	MaxConnection       int    `config:"max_connection"`
	MaxResponseBodySize int    `config:"max_response_size"`

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