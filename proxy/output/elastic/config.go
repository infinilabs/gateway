// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package elastic

import "time"

type ProxyConfig struct {
	Elasticsearch string `config:"elasticsearch"`
	Balancer      string `config:"balancer"`

	MaxConnection                     int  `config:"max_connection_per_node"`
	MaxResponseBodySize               int  `config:"max_response_size"`
	MaxRetryTimes                     int  `config:"max_retry_times"`
	RetryOnBackendFailure             bool `config:"retry_on_backend_failure"`
	RetryReadonlyOnlyOnBackendFailure bool `config:"retry_readonly_on_backend_failure"` //usually it is safety to retry readonly requests, GET/HEAD verbs only, as write may have partial failure, retry may cause duplicated writes
	RetryWriteOpsOnBackendFailure     bool `config:"retry_writes_on_backend_failure"`   //POST/PUT/PATCH requests, which means may not good for retry, but you can sill opt it on, and it is preferred to work with other flow/filters
	RetryOnBackendBusy                bool `config:"retry_on_backend_busy"`
	RetryDelayInMs                    int  `config:"retry_delay_in_ms"`

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
