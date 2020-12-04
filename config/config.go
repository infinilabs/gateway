package config

type UpstreamConfig struct {
	Name   string `config:"name"`
	Weight int    `config:"weight"`
	//QueueName     string `config:"queue_name"`
	//MaxQueueDepth int64  `config:"max_queue_depth"`
	MaxConnection int    `config:"max_connection"`
	Enabled       bool   `config:"enabled"`
	Writeable     bool   `config:"writeable"`
	Readable      bool   `config:"readable"`
	Timeout       string `config:"timeout"`
	Elasticsearch string `config:"elasticsearch"`
}

//func (v *UpstreamConfig) SafeGetQueueName() string {
//	queueName := v.QueueName
//	if queueName == "" {
//		queueName = v.Name
//	}
//	return queueName
//}

type ProxyConfig struct {
	Enabled       bool   `config:"enabled"`
	Elasticsearch string `config:"elasticsearch"`
	//Upstream     []UpstreamConfig `config:"upstream"`
	Balancer     string   `config:"balancer"`
	//PassPatterns []string `config:"pass_through"`
	Timeout      string   `config:"timeout"`

	//TracingEnabled bool `config:"tracing_enabled"`

	MaxConnection       int `config:"max_connection"`
	MaxResponseBodySize int `config:"max_response_size"`

	Weights  map[string]int `config:"weight"`
	Discover struct {
		Enabled    bool     `config:"enabled"`
		NodeFilter []string `config:"node_filter"`
	} `config:"discovery"`

	//AsyncWrite    bool                 `config:"async_write"`
	//TLSConfig     config.TLSConfig     `config:"tls"`
	//NetworkConfig config.NetworkConfig `config:"network"`
	//CacheConfig   CacheConfig          `config:"cache"`
}

type CacheConfig struct {
	Enabled        bool   `config:"enabled"`
	Type           string `config:"type"` //redis,local
	TTL            string `config:"ttl"`
	AsyncSearchTTL string `config:"async_search_ttl"`
	MaxCachedItem  int64  `config:"max_cached_item"`
	//generalTTLDuration     time.Duration
	//asyncSearchTTLDuration time.Duration
}

//
//func (config CacheConfig) GetChaosTTLDuration() time.Duration {
//	baseTTL := config.GetTTLDuration().Milliseconds()
//	randomTTL := rand.Int63n(baseTTL / 5)
//	return (time.Duration(baseTTL + randomTTL)) * time.Millisecond
//}
//
//func (config CacheConfig) GetTTLDuration() time.Duration {
//	if config.generalTTLDuration > 0 {
//		return config.generalTTLDuration
//	}
//
//	if config.TTL != "" {
//		dur, err := time.ParseDuration(config.TTL)
//		if err != nil {
//			dur, _ = time.ParseDuration("10s")
//		}
//		config.generalTTLDuration = dur
//	} else {
//		config.generalTTLDuration = time.Second * 10
//	}
//	return config.generalTTLDuration
//}
//
//func (config CacheConfig) GetAsyncSearchTTLDuration() time.Duration {
//	if config.asyncSearchTTLDuration > 0 {
//		return config.asyncSearchTTLDuration
//	}
//
//	if config.AsyncSearchTTL != "" {
//		dur, err := time.ParseDuration(config.AsyncSearchTTL)
//		if err != nil {
//			dur, _ = time.ParseDuration("30m")
//		}
//		config.asyncSearchTTLDuration = dur
//	} else {
//		config.asyncSearchTTLDuration = time.Minute * 30
//	}
//	return config.asyncSearchTTLDuration
//}
//
//const Url pipeline.ParaKey = "url"
//const Method pipeline.ParaKey = "method"
//const Body pipeline.ParaKey = "body"
//const Upstream pipeline.ParaKey = "upstream"
//const Response pipeline.ParaKey = "response"
//const ResponseSize pipeline.ParaKey = "response_size"
//const ResponseStatusCode pipeline.ParaKey = "response_code"
//const Message pipeline.ParaKey = "message"
//
////Bucket
//const InactiveUpstream = "inactive_upstream"
//
//var proxyConfig ProxyConfig
//
//var upstreams map[string]UpstreamConfig = map[string]UpstreamConfig{}
//
//var l sync.RWMutex
//
//func GetUpstreamConfig(key string) UpstreamConfig {
//	l.RLock()
//	defer l.RUnlock()
//	v := upstreams[key]
//	return v
//}
//
//func GetUpstreamConfigs() map[string]UpstreamConfig {
//	return upstreams
//}
//
//func GetActiveUpstreamConfigs() map[string]UpstreamConfig {
//	active := map[string]UpstreamConfig{}
//	for k, v := range upstreams {
//		if v.Enabled {
//			active[k] = v
//		}
//	}
//	return active
//}
//
//func UpdateUpstreamWriteableStatus(key string, active bool) {
//	l.Lock()
//	defer l.Unlock()
//	v := upstreams[key]
//	v.Writeable = active
//	upstreams[key] = v
//}
//
//func UpdateUpstreamReadableStatus(key string, active bool) {
//	l.Lock()
//	defer l.Unlock()
//	v := upstreams[key]
//	v.Readable = active
//	upstreams[key] = v
//}
//
//func GetProxyConfig() ProxyConfig {
//	return proxyConfig
//}
//
//func SetProxyConfig(cfg ProxyConfig) {
//	proxyConfig = cfg
//	SetUpstream(cfg.Upstream)
//}
//
//func SetUpstream(ups []UpstreamConfig) {
//	l.Lock()
//	defer l.Unlock()
//	for _, v := range ups {
//		//default Active is true
//		v.Writeable = true
//		v.Readable = true
//
//		//TODO get upstream status from DB, override active field
//		upstreams[v.Name] = v
//	}
//}
