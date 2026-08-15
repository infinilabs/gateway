/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package otlp

import (
	"context"

	logsv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	grpc "google.golang.org/grpc"

	"infini.sh/framework/core/global"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/otel"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	otlpcodec "infini.sh/framework/plugins/otlp"
)

// logServer implements the OTLP LogsService.
type logServer struct {
	logsv1.UnimplementedLogsServiceServer

	queueName string
}

// RegisterLogsService wires the OTLP LogsService handler into a gRPC server.
func RegisterLogsService(server *grpc.Server, queueName string) {
	logsv1.RegisterLogsServiceServer(server, &logServer{queueName: queueName})
}

// Export implements logsv1.LogsServiceServer.
func (s *logServer) Export(ctx context.Context, req *logsv1.ExportLogsServiceRequest) (*logsv1.ExportLogsServiceResponse, error) {
	events := otlpcodec.RecordsFromExportRequest(req)
	if len(events) == 0 {
		return &logsv1.ExportLogsServiceResponse{}, nil
	}

	qConfig := queue.GetOrInitConfig(s.queueName)
	pushed := 0
	for _, e := range events {
		data, err := otel.EncodeEnvelope(e)
		if err != nil {
			log.Warnf("otlp intake: failed to encode record: %v", err)
			stats.Increment("otlp", "encode_error")
			continue
		}
		if err := queue.Push(qConfig, data); err != nil {
			log.Errorf("otlp intake: failed to enqueue record: %v", err)
			stats.Increment("otlp", "push_error")
			continue
		}
		pushed++
	}
	stats.IncrementBy("otlp", "records", int64(pushed))
	if global.Env().IsDebug {
		log.Debugf("otlp intake: accepted %d records onto queue [%v]", pushed, s.queueName)
	}
	return &logsv1.ExportLogsServiceResponse{}, nil
}
