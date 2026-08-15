/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package otlp

// DefaultQueueName is the queue that feeds the gateway's log processing
// pipeline when none is configured.
const DefaultQueueName = "log_processing"

// Config of the gateway OTLP intake module.
type Config struct {
	Enabled              bool   `config:"enabled"`
	Bind                 string `config:"bind"`
	Queue                string `config:"queue"`
	MaxReceiveMessageSize int64 `config:"max_receive_message_size"`
}
