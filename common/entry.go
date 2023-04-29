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
	SkipReduceMemoryUsage   bool   `config:"skip_reduce_memory" json:"skip_reduce_memory,omitempty" elastic_mapping:"skip_reduce_memory: { type: boolean }"`
	ReadTimeout         int    `config:"read_timeout" json:"read_timeout,omitempty" elastic_mapping:"read_timeout: { type: integer }"`
	WriteTimeout        int    `config:"write_timeout" json:"write_timeout,omitempty" elastic_mapping:"write_timeout: { type: integer }"`
	DisableTCPKeepalive        bool   `config:"disable_tcp_keepalive" json:"disable_tcp_keepalive,omitempty" elastic_mapping:"disable_tcp_keepalive: { type: boolean }"`
	TCPKeepaliveSeconds int    `config:"tcp_keepalive_in_seconds" json:"tcp_keepalive_in_seconds,omitempty" elastic_mapping:"tcp_keepalive_in_seconds: { type: integer }"`
	IdleTimeout         int    `config:"idle_timeout" json:"idle_timeout,omitempty" elastic_mapping:"idle_timeout: { type: integer }"`

	MaxIdleWorkerDurationSeconds       int `config:"max_idle_worker_duration_in_seconds" json:"max_idle_worker_duration_in_seconds,omitempty" elastic_mapping:"max_idle_worker_duration_in_seconds: { type: integer }"`
	SleepWhenConcurrencyLimitsExceeded int `config:"sleep_when_concurrency_limits_exceeded_in_seconds" json:"sleep_when_concurrency_limits_exceeded_in_seconds,omitempty" elastic_mapping:"sleep_when_concurrency_limits_exceeded_in_seconds: { type: integer }"`

	ReadBufferSize  int `config:"read_buffer_size" json:"read_buffer_size,omitempty" elastic_mapping:"read_buffer_size: { type: integer }"`
	WriteBufferSize int `config:"write_buffer_size" json:"write_buffer_size,omitempty" elastic_mapping:"write_buffer_size: { type: integer }"`

	MaxRequestBodySize int `config:"max_request_body_size" json:"max_request_body_size,omitempty" elastic_mapping:"max_request_body_size: { type: integer }"`
	MaxConcurrency     int `config:"max_concurrency" json:"max_concurrency,omitempty" elastic_mapping:"max_concurrency: { type: integer }"`
	MaxConnsPerIP      int `config:"max_conns_per_ip" json:"max_conns_per_ip,omitempty" elastic_mapping:"max_conns_per_ip: { type: integer }"`

	TLSConfig        config.TLSConfig     `config:"tls" json:"tls,omitempty" elastic_mapping:"tls: { type: object }"`
	NetworkConfig    config.NetworkConfig `config:"network" json:"network,omitempty" elastic_mapping:"network: { type: object }"`
	RouterConfigName string               `config:"router" json:"router,omitempty" elastic_mapping:"router: { type: keyword }"`
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
	Enabled      bool    `config:"enabled" json:"enabled,omitempty" elastic_mapping:"enabled: { type: boolean }"`
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


	RuleToggleEnabled      bool    `config:"rule_toggle_enabled" json:"rule_toggle_enabled,omitempty" elastic_mapping:"rule_toggle_enabled: { type: boolean }"`
	Rules              []RuleConfig `config:"rules" json:"rules,omitempty" elastic_mapping:"rules: { type: object }"`

	IPAccessRules IPAccessRules `config:"ip_access_control" json:"ip_access_rules,omitempty" elastic_mapping:"ip_access_rules: { type: object }"`
}

type IPAccessRules struct {
	Enabled             bool   `config:"enabled" json:"enabled,omitempty" elastic_mapping:"enabled: { type: boolean }"`
	ClientIP struct{
		DeniedList []string     `config:"denied" json:"denied,omitempty" elastic_mapping:"denied: { type: keyword }"`
		PermittedList []string     `config:"permitted" json:"permitted,omitempty" elastic_mapping:"permitted: { type: keyword }"`
	} `config:"client_ip" json:"client_ip,omitempty" elastic_mapping:"client_ip: { type: object }"`
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
