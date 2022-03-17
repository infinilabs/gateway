/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/orm"
)

type EntryConfig struct {
	orm.ORMObjectBase

	Name                string `config:"name" json:"name,omitempty" elastic_mapping:"name:{type:keyword,fields:{text: {type: text}}}"`
	Enabled             bool   `config:"enabled" json:"enabled,omitempty" elastic_mapping:"enabled: { type: boolean }"`
	DirtyShutdown       bool   `config:"dirty_shutdown" json:"dirty_shutdown,omitempty" elastic_mapping:"dirty_shutdown: { type: boolean }"`
	ReduceMemoryUsage   bool   `config:"reduce_memory_usage" json:"reduce_memory_usage,omitempty" elastic_mapping:"reduce_memory_usage: { type: boolean }"`
	ReadTimeout         int    `config:"read_timeout" json:"read_timeout,omitempty" elastic_mapping:"read_timeout: { type: integer }"`
	WriteTimeout        int    `config:"write_timeout" json:"write_timeout,omitempty" elastic_mapping:"write_timeout: { type: integer }"`
	TCPKeepalive        bool   `config:"tcp_keepalive" json:"tcp_keepalive,omitempty" elastic_mapping:"tcp_keepalive: { type: boolean }"`
	TCPKeepaliveSeconds int    `config:"tcp_keepalive_in_seconds" json:"tcp_keepalive_in_seconds,omitempty" elastic_mapping:"tcp_keepalive_in_seconds: { type: integer }"`
	IdleTimeout         int    `config:"idle_timeout" json:"idle_timeout,omitempty" elastic_mapping:"idle_timeout: { type: integer }"`

	MaxIdleWorkerDurationSeconds int `config:"max_idle_worker_duration_in_seconds" json:"max_idle_worker_duration_in_seconds,omitempty" elastic_mapping:"max_idle_worker_duration_in_seconds: { type: integer }"`

	ReadBufferSize  int `config:"read_buffer_size" json:"read_buffer_size,omitempty" elastic_mapping:"read_buffer_size: { type: integer }"`
	WriteBufferSize int `config:"write_buffer_size" json:"write_buffer_size,omitempty" elastic_mapping:"write_buffer_size: { type: integer }"`

	MaxRequestBodySize int                  `config:"max_request_body_size" json:"max_request_body_size,omitempty" elastic_mapping:"max_request_body_size: { type: integer }"`
	MaxConcurrency     int                  `config:"max_concurrency" json:"max_concurrency,omitempty" elastic_mapping:"max_concurrency: { type: integer }"`
	TLSConfig          config.TLSConfig     `config:"tls" json:"tls,omitempty" elastic_mapping:"tls: { type: object }"`
	NetworkConfig      config.NetworkConfig `config:"network" json:"network,omitempty" elastic_mapping:"network: { type: object }"`
	RouterConfigName   string               `config:"router" json:"router,omitempty" elastic_mapping:"router: { type: keyword }"`
}

func (this *EntryConfig) Equals(target *EntryConfig) bool {
	if this.Enabled != target.Enabled ||
		this.DirtyShutdown != target.DirtyShutdown ||
		this.RouterConfigName != target.RouterConfigName ||
		this.TLSConfig.TLSEnabled != target.TLSConfig.TLSEnabled ||
		this.NetworkConfig.GetBindingAddr() != target.NetworkConfig.GetBindingAddr() {
		return false
	}
	return true
}

type RuleConfig struct {
	Method      []string `config:"method" json:"method,omitempty"      elastic_mapping:"method: { type: keyword }"`
	PathPattern []string `config:"pattern" json:"pattern,omitempty"      elastic_mapping:"pattern: { type: keyword }"`
	Flow        []string `config:"flow" json:"flow,omitempty"      elastic_mapping:"flow: { type: keyword }"`
	Description string   `config:"description" json:"description,omitempty"      elastic_mapping:"description: { type: keyword }"`
}

type FilterConfig struct {
	ID         string                 `config:"id"`
	Name       string                 `config:"name"`
	Parameters map[string]interface{} `config:"parameters"`
}

type RouterConfig struct {
	orm.ORMObjectBase

	Name        string `config:"name" json:"name,omitempty" elastic_mapping:"name:{type:keyword,fields:{text: {type: text}}}"`
	DefaultFlow string `config:"default_flow" json:"default_flow,omitempty" elastic_mapping:"default_flow: { type: keyword }"`
	TracingFlow string `config:"tracing_flow" json:"tracing_flow,omitempty" elastic_mapping:"tracing_flow: { type: keyword }"`

	Rules              []RuleConfig `config:"rules" json:"rules,omitempty" elastic_mapping:"rules: { type: object }"`
	DeniedClientIPList []string     `config:"denied_client_ip_list" json:"denied_client_ip_list,omitempty" elastic_mapping:"denied_client_ip_list: { type: keyword }"`
}

type FlowConfig struct {
	orm.ORMObjectBase

	Name        string                   `config:"name" json:"name,omitempty" elastic_mapping:"name:{type:keyword,fields:{text: {type: text}}}"`
	Filters     []*config.Config         `config:"filter" json:"-"`
	JsonFilters []map[string]interface{} `json:"filter,omitempty"`
}

func (flow *FlowConfig) GetConfig() []*config.Config {

	for _, v := range flow.JsonFilters {
		c, err := config.NewConfigFrom(v)
		if err != nil {
			panic(err)
		}
		flow.Filters = append(flow.Filters, c)
	}
	return flow.Filters
}
