/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"time"
)

type EntryConfig struct {
	//common properties start
	Id      string    `json:"id,omitempty"      elastic_meta:"_id" elastic_mapping:"id: { type: keyword }"`
	Created time.Time `json:"created,omitempty" elastic_mapping:"created: { type: date }"`
	Updated time.Time `json:"updated,omitempty" elastic_mapping:"updated: { type: date }"`
	//common properties end

	Name                string `config:"name" json:"name,omitempty" elastic_mapping:"name:{type:keyword,fields:{text: {type: text}}}"`
	Enabled             bool   `config:"enabled" json:"enabled,omitempty" elastic_mapping:"enabled: { type: boolean }"`
	DirtyShutdown       bool   `config:"dirty_shutdown" json:"dirty_shutdown,omitempty" elastic_mapping:"dirty_shutdown: { type: boolean }"`
	ReduceMemoryUsage   bool   `config:"reduce_memory_usage" json:"reduce_memory_usage,omitempty" elastic_mapping:"reduce_memory_usage: { type: boolean }"`
	ReadTimeout         int    `config:"read_timeout" json:"read_timeout,omitempty" elastic_mapping:"read_timeout: { type: integer }"`
	WriteTimeout        int    `config:"write_timeout" json:"write_timeout,omitempty" elastic_mapping:"write_timeout: { type: integer }"`
	TCPKeepalive        bool   `config:"tcp_keepalive" json:"tcp_keepalive,omitempty" elastic_mapping:"tcp_keepalive: { type: boolean }"`
	TCPKeepaliveSeconds int    `config:"tcp_keepalive_in_seconds" json:"tcp_keepalive_in_seconds,omitempty" elastic_mapping:"tcp_keepalive_in_seconds: { type: integer }"`
	IdleTimeout         int    `config:"idle_timeout" json:"idle_timeout,omitempty" elastic_mapping:"idle_timeout: { type: integer }"`

	ReadBufferSize  int `config:"read_buffer_size" json:"read_buffer_size,omitempty" elastic_mapping:"read_buffer_size: { type: integer }"`
	WriteBufferSize int `config:"write_buffer_size" json:"write_buffer_size,omitempty" elastic_mapping:"write_buffer_size: { type: integer }"`

	MaxRequestBodySize int                  `config:"max_request_body_size" json:"max_request_body_size,omitempty" elastic_mapping:"max_request_body_size: { type: integer }"`
	MaxConcurrency     int                  `config:"max_concurrency" json:"max_concurrency,omitempty" elastic_mapping:"max_concurrency: { type: integer }"`
	TLSConfig          config.TLSConfig     `config:"tls" json:"tls,omitempty" elastic_mapping:"tls: { type: object }"`
	NetworkConfig      config.NetworkConfig `config:"network" json:"network,omitempty" elastic_mapping:"network: { type: object }"`
	RouterConfigName   string               `config:"router" json:"router,omitempty" elastic_mapping:"router: { type: keyword }"`
}

type RuleConfig struct {
	ID          string   `config:"id"`
	Description string   `config:"desc"`
	Method      []string `config:"method"`
	PathPattern []string `config:"pattern"`
	Flow        []string `config:"flow"`
}

type FilterConfig struct {
	ID         string                 `config:"id"`
	Name       string                 `config:"name"`
	Parameters map[string]interface{} `config:"parameters"`
}

type RouterConfig struct {
	Name        string       `config:"name"`
	DefaultFlow string       `config:"default_flow"`
	Rules       []RuleConfig `config:"rules"`
	TracingFlow string       `config:"tracing_flow"`
}

type FlowConfig struct {
	Name      string                `config:"name"`
	Filters   []FilterConfig        `config:"filter_v1"`
	FiltersV2 pipeline.PluginConfig `config:"filter"`
}

type FilterPropertie struct {
	Type         string      `config:"type" json:"type"`
	SubType      string      `config:"sub_type" json:"sub_type"`
	DefaultValue interface{} `config:"default_value" json:"default_value"`
}
