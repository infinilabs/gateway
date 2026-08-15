/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package otlp

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	logsv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsdata "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/otel"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	memqueue "infini.sh/framework/modules/queue/mem_queue"
)

// memKV is a minimal in-memory kv.KVStore so queue.GetOrInitConfig works
// without booting the whole framework in tests.
type memKV struct {
	mu     sync.RWMutex
	bucket map[string]map[string][]byte
}

func (k *memKV) Open() error  { return nil }
func (k *memKV) Close() error { return nil }
func (k *memKV) ExistsKey(bucket string, key []byte) (bool, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	_, ok := k.bucket[bucket][string(key)]
	return ok, nil
}
func (k *memKV) DeleteKey(bucket string, key []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.bucket[bucket], string(key))
	return nil
}
func (k *memKV) GetValue(bucket string, key []byte) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.bucket[bucket][string(key)], nil
}
func (k *memKV) GetCompressedValue(bucket string, key []byte) ([]byte, error) {
	return k.GetValue(bucket, key)
}
func (k *memKV) AddValue(bucket string, key []byte, value []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.bucket == nil {
		k.bucket = map[string]map[string][]byte{}
	}
	if k.bucket[bucket] == nil {
		k.bucket[bucket] = map[string][]byte{}
	}
	k.bucket[bucket][string(key)] = value
	return nil
}
func (k *memKV) AddValueCompress(bucket string, key []byte, value []byte) error {
	return k.AddValue(bucket, key, value)
}

// memQueueAdapter fills in the few QueueAPI methods MemoryQueue lacks.
type memQueueAdapter struct {
	*memqueue.MemoryQueue
}

func (a *memQueueAdapter) Destroy(string) error         { return nil }
func (a *memQueueAdapter) GetStorageSize(string) uint64 { return 0 }

func setupTestQueue(t *testing.T) {
	t.Helper()
	kv.Register("test_mem_kv", &memKV{})
	mem := &memQueueAdapter{MemoryQueue: &memqueue.MemoryQueue{}}
	mem.Setup()
	queue.Register("memory", mem)
	queue.RegisterDefaultHandler(mem)
}

// TestExportEndToEnd spins the OTLP intake on an ephemeral port, exports
// one record through a real gRPC client and verifies that the decoded
// otel envelope lands on the downstream queue.
func TestExportEndToEnd(t *testing.T) {
	setupTestQueue(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	server := grpc.NewServer()
	RegisterLogsService(server, "otlp_test_queue")
	go server.Serve(ln)
	defer server.Stop()

	conn, err := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Now().UTC()
	req := &logsv1.ExportLogsServiceRequest{
		ResourceLogs: []*logsdata.ResourceLogs{{
			Resource: &resourcev1.Resource{
				Attributes: []*commonv1.KeyValue{
					{Key: otel.ResourceHostName, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "web-1"}}},
				},
			},
			ScopeLogs: []*logsdata.ScopeLogs{{
				LogRecords: []*logsdata.LogRecord{{
					TimeUnixNano:   uint64(now.UnixNano()),
					SeverityText:   "ERROR",
					SeverityNumber: logsdata.SeverityNumber_SEVERITY_NUMBER_ERROR,
					Body:           &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "connection refused"}},
					Attributes: []*commonv1.KeyValue{
						{Key: "service_name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "payment-svc"}}},
					},
				}},
			}},
		}},
	}

	client := logsv1.NewLogsServiceClient(conn)
	resp, err := client.Export(context.Background(), req)
	if err != nil {
		t.Fatalf("export rpc: %v", err)
	}
	if resp == nil {
		t.Fatal("nil export response")
	}

	// drain the queue and verify the envelope
	data, err := queue.Pop(queue.GetOrInitConfig("otlp_test_queue"))
	if err != nil {
		t.Fatalf("pop: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("no message landed on the queue")
	}

	rec, err := otel.DecodeEnvelope(data)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if rec.Fields[otel.FieldLogLevel] != "ERROR" {
		t.Fatalf("log_level = %v", rec.Fields[otel.FieldLogLevel])
	}
	if rec.Fields[otel.FieldMessage] != "connection refused" {
		t.Fatalf("message = %v", rec.Fields[otel.FieldMessage])
	}
	if rec.Fields["service_name"] != "payment-svc" {
		t.Fatalf("service_name = %v", rec.Fields["service_name"])
	}
	res, ok := rec.Meta[otel.MetaResourceKey].(util.MapStr)
	if !ok || res[otel.ResourceHostName] != "web-1" {
		t.Fatalf("resource = %v", rec.Meta[otel.MetaResourceKey])
	}
}
