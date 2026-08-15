/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package otlp is the gateway's OTLP/gRPC intake module. It exposes the
// standard OpenTelemetry LogsService (Export) endpoint so that INFINI
// agents — or any OTel SDK / Collector — can push logs into the gateway
// processing pipeline.
//
// Ingested records are decoded into framework events (otel Log Data
// Model, snake_case attributes) and appended to a local queue for the
// downstream consumer → processors → sinks chain.
//
// Configuration (gateway.yml):
//
//	otlp:
//	  enabled: true
//	  bind: ":4317"          # standard OTLP/gRPC port
//	  queue: log_processing  # queue that feeds the processing pipeline
//	  max_receive_message_size: 16mb
package otlp

import (
	"net"
	"strings"

	grpc "google.golang.org/grpc"

	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/module"
)

// Module implements module.Module.
type Module struct {
	server   *grpc.Server
	listener net.Listener
}

// Name implements module.Module; the config section must match.
func (m *Module) Name() string { return "otlp" }

var moduleConfig = Config{}

// Setup implements module.Module.
func (m *Module) Setup() {
	exists, err := env.ParseConfig("otlp", &moduleConfig)
	if exists && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}
	if moduleConfig.Bind == "" {
		moduleConfig.Bind = ":4317"
	}
	if moduleConfig.Queue == "" {
		moduleConfig.Queue = DefaultQueueName
	}
}

// Start implements module.Module; it listens and serves on a background
// goroutine so it never blocks the module lifecycle.
func (m *Module) Start() error {
	ln, err := net.Listen("tcp", moduleConfig.Bind)
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{}
	if moduleConfig.MaxReceiveMessageSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(int(moduleConfig.MaxReceiveMessageSize)))
	}

	server := grpc.NewServer(opts...)
	RegisterLogsService(server, moduleConfig.Queue)

	m.server = server
	m.listener = ln
	log.Infof("otlp intake listening on %v, downstream queue [%v]", moduleConfig.Bind, moduleConfig.Queue)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("otlp intake server panicked: %v", r)
			}
		}()
		if err := server.Serve(ln); err != nil && !strings.Contains(err.Error(), "closed") {
			log.Errorf("otlp intake server exited: %v", err)
		}
	}()
	return nil
}

// Stop implements module.Module.
func (m *Module) Stop() error {
	if m.server != nil {
		m.server.GracefulStop()
		m.server = nil
	}
	if m.listener != nil {
		_ = m.listener.Close()
	}
	return nil
}

// ensure the module interface is satisfied
var _ module.Module = (*Module)(nil)
