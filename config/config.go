package config

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"sync"
	"time"
)

type UpstreamConfig struct {
	Name          string `config:"name"`
	Weight        int    `config:"weight"`
	QueueName     string `config:"queue_name"`
	MaxQueueDepth int64  `config:"max_queue_depth"`
	MaxConnection int    `config:"max_connection"`
	Enabled       bool   `config:"enabled"`
	Writeable     bool   `config:"writeable"`
	Readable      bool   `config:"readable"`
	Timeout       string `config:"timeout"`
	Elasticsearch string `config:"elasticsearch"`
}

func (v *UpstreamConfig) SafeGetQueueName() string {
	queueName := v.QueueName
	if queueName == "" {
		queueName = v.Name
	}
	return queueName
}

type ProxyConfig struct {
	Upstream            []UpstreamConfig     `config:"upstream"`
	Balancer            string               `config:"balancer"`
	PassthroughPatterns []string             `config:"pass_through"`
	Enabled             bool                 `config:"enabled"`
	TLSConfig           config.TLSConfig     `config:"tls"`
	NetworkConfig       config.NetworkConfig `config:"network"`
	CacheConfig         CacheConfig          `config:"cache"`
}

type CacheConfig struct {
	Enabled       bool   `config:"enabled"`
	TTL           string `config:"ttl"`
	MaxCachedItem int64  `config:"max_cached_item"`
	duration      time.Duration
}

func (config CacheConfig) GetTTLDuration() time.Duration {
	if config.duration > 0 {
		return config.duration
	}

	if config.TTL != "" {
		dur, err := time.ParseDuration(config.TTL)
		if err != nil {
			dur, _ = time.ParseDuration("10s")
		}
		config.duration = dur
	}
	return config.duration
}

const Url pipeline.ParaKey = "url"
const Method pipeline.ParaKey = "method"
const Body pipeline.ParaKey = "body"
const Upstream pipeline.ParaKey = "upstream"
const Response pipeline.ParaKey = "response"
const ResponseSize pipeline.ParaKey = "response_size"
const ResponseStatusCode pipeline.ParaKey = "response_code"
const Message pipeline.ParaKey = "message"

//Bucket
const InactiveUpstream = "inactive_upstream"

var proxyConfig ProxyConfig

var upstreams map[string]UpstreamConfig = map[string]UpstreamConfig{}

var l sync.RWMutex

func GetUpstreamConfig(key string) UpstreamConfig {
	l.RLock()
	defer l.RUnlock()
	v := upstreams[key]
	return v
}

func GetUpstreamConfigs() map[string]UpstreamConfig {
	return upstreams
}

func GetActiveUpstreamConfigs() map[string]UpstreamConfig {
	active := map[string]UpstreamConfig{}
	for k, v := range upstreams {
		if v.Enabled {
			active[k] = v
		}
	}
	return active
}

func UpdateUpstreamWriteableStatus(key string, active bool) {
	l.Lock()
	defer l.Unlock()
	v := upstreams[key]
	v.Writeable = active
	upstreams[key] = v
}

func UpdateUpstreamReadableStatus(key string, active bool) {
	l.Lock()
	defer l.Unlock()
	v := upstreams[key]
	v.Readable = active
	upstreams[key] = v
}

func GetProxyConfig() ProxyConfig {
	return proxyConfig
}

func SetProxyConfig(cfg ProxyConfig) {
	proxyConfig = cfg
	SetUpstream(cfg.Upstream)
}

func SetUpstream(ups []UpstreamConfig) {
	l.Lock()
	defer l.Unlock()
	for _, v := range ups {
		//default Active is true
		v.Writeable = true
		v.Readable = true

		//TODO get upstream status from DB, override active field
		upstreams[v.Name] = v
	}
}
