package es_scroll

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"infini.sh/framework/core/elastic"
	"infini.sh/framework/lib/fasthttp"
)

func TestBuildBulkMetaLineEscapesSpecialCharacters(t *testing.T) {
	line := buildBulkMetaLine("index", "\x1enginx_zstd-6", `_doc"v2`, "doc-\n1", `route"\test`)

	if !json.Valid(line) {
		t.Fatalf("expected valid json line, got %q", string(line))
	}

	var got map[string]map[string]string
	if err := json.Unmarshal(line, &got); err != nil {
		t.Fatalf("failed to unmarshal line: %v", err)
	}

	meta := got["index"]
	if meta["_index"] != "\x1enginx_zstd-6" {
		t.Fatalf("unexpected _index: %q", meta["_index"])
	}
	if meta["_type"] != `_doc"v2` {
		t.Fatalf("unexpected _type: %q", meta["_type"])
	}
	if meta["_id"] != "doc-\n1" {
		t.Fatalf("unexpected _id: %q", meta["_id"])
	}
	if meta["routing"] != `route"\test` {
		t.Fatalf("unexpected routing: %q", meta["routing"])
	}
}

func TestTruncateLogValue(t *testing.T) {
	got := truncateLogValue("abcdefghijklmnopqrstuvwxyz", 8)
	if got != "abcdefgh..." {
		t.Fatalf("unexpected truncated value: %q", got)
	}
}

func TestWrapScrollRequestErrorIncludesContext(t *testing.T) {
	processor := &ScrollProcessor{
		config: Config{
			Elasticsearch: "source-cluster",
			Indices:       "logs-*",
			SliceSize:     4,
			BatchSize:     500,
			ScrollTime:    "10m",
			QueryString:   "status:500",
		},
		clientID:       "source-cluster",
		requestTimeout: 5 * time.Second,
	}

	req := &fasthttp.Request{}
	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1:9200/_search/scroll?scroll=10m&scroll_id=DXF1ZXJ5QW5kRmV0Y2gBAAAAAAA")
	res := &fasthttp.Response{}

	apiCtx := &elastic.APIContext{
		Request:  req,
		Response: res,
	}

	err := processor.wrapScrollRequestError(
		"next scroll",
		2,
		errors.New("context deadline exceeded"),
		nil,
		nil,
		"DXF1ZXJ5QW5kRmV0Y2gBAAAAAAA",
		apiCtx,
	)

	msg := err.Error()
	wantParts := []string{
		"next scroll failed",
		"cluster=source-cluster",
		"indices=logs-*",
		"slice=2/4",
		"scroll=10m",
		"batch_size=500",
		"request_timeout=5s",
		"method=POST",
		"host=127.0.0.1:9200",
		"request_uri=/_search/scroll?scroll=10m&scroll_id=DXF1ZXJ5QW5kRmV0Y2gBAAAAAAA",
		"scroll_id_prefix=DXF1ZXJ5QW5kRmV0Y2gBAAAAAAA",
		"query_string=status:500",
		"response=<empty>",
		"context deadline exceeded",
	}

	for _, want := range wantParts {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in error, got %q", want, msg)
		}
	}
}

func TestWrapQueuePushErrorIncludesContext(t *testing.T) {
	processor := &ScrollProcessor{
		config: Config{
			Elasticsearch: "source-cluster",
			Indices:       ".infini_metrics-00001",
		},
	}

	err := processor.wrapQueuePushError("d7tl92b0ebiths0ri8500", 3, 65536, errors.New("operation timed out: context deadline exceeded"))
	msg := err.Error()

	wantParts := []string{
		"push scroll batch to queue failed",
		"cluster=source-cluster",
		"indices=.infini_metrics-00001",
		"queue=d7tl92b0ebiths0ri8500",
		"partition=3",
		"payload_bytes=65536",
		"operation timed out: context deadline exceeded",
	}
	for _, want := range wantParts {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in error, got %q", want, msg)
		}
	}
}

func TestSplitBulkPayloadByBytesPreservesOperations(t *testing.T) {
	var payload []byte
	for i := 0; i < 3; i++ {
		payload = append(payload, buildBulkMetaLine("index", "logs-test", "_doc", "doc-"+string(rune('1'+i)), "")...)
		payload = append(payload, []byte(fmt.Sprintf("{\"message\":\"doc-%d\"}\n", i+1))...)
	}

	chunks, err := splitBulkPayloadByBytes(payload, len(payload)/2)
	if err != nil {
		t.Fatalf("splitBulkPayloadByBytes returned error: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected payload to be split, got %d chunk(s)", len(chunks))
	}

	totalOps := 0
	for _, chunk := range chunks {
		if len(chunk) > len(payload)/2 {
			t.Fatalf("chunk too large: %d", len(chunk))
		}

		ops, err := elastic.WalkBulkRequests("", chunk, nil,
			func(metaBytes []byte, actionStr, index, typeName, id, routing string, offset int) error { return nil },
			func(payloadBytes []byte, actionStr, index, typeName, id, routing string) {},
			nil,
		)
		if err != nil {
			t.Fatalf("chunk should remain valid bulk payload: %v", err)
		}
		totalOps += ops
	}

	if totalOps != 3 {
		t.Fatalf("expected 3 operations after splitting, got %d", totalOps)
	}
}

func TestEffectiveScrollRequestTimeoutUsesMinimum(t *testing.T) {
	timeout := effectiveScrollRequestTimeout(&elastic.ElasticsearchConfig{RequestTimeout: 5})
	if timeout != minScrollRequestTimeout {
		t.Fatalf("unexpected timeout: got %s want %s", timeout, minScrollRequestTimeout)
	}
}

func TestHasOversizedBulkOperation(t *testing.T) {
	meta := buildBulkMetaLine("index", "logs-test", "_doc", "doc-1", "")
	payload := buildBulkOperationBytes(meta[:len(meta)-1], []byte(`{"message":"`+strings.Repeat("x", maxQueuePayloadBytes)+`"}`))

	oversized, maxOperationBytes, err := hasOversizedBulkOperation(payload, maxQueuePayloadBytes)
	if err != nil {
		t.Fatalf("hasOversizedBulkOperation returned error: %v", err)
	}
	if !oversized {
		t.Fatal("expected bulk operation to be oversized")
	}
	if maxOperationBytes <= maxQueuePayloadBytes {
		t.Fatalf("expected oversized operation bytes, got %d", maxOperationBytes)
	}
}
