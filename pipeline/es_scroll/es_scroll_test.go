package es_scroll

import (
	"errors"
	"strings"
	"testing"
	"time"

	"infini.sh/framework/core/elastic"
	"infini.sh/framework/lib/fasthttp"
)

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
