/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package config

import "infini.sh/framework/core/config"

type EntryConfig struct {
	Enabled        bool                 `config:"enabled"`
	Name           string               `config:"name"`
	MaxConcurrency int                  `config:"max_concurrency"`
	TLSConfig      config.TLSConfig     `config:"tls"`
	NetworkConfig  config.NetworkConfig `config:"network"`
}
