package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestLoggerFromBindsRequestAndTraceWithoutDuplicateKeys(t *testing.T) {
	var output bytes.Buffer
	ctx := WithRequestID(context.Background(), "request-123")
	ctx = withRequestLogger(ctx, NewLogger(&output, slog.LevelInfo))
	tid, _ := trace.TraceIDFromHex("01234567890123456789012345678901")
	sid, _ := trace.SpanIDFromHex("0123456789012345")
	ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid}))
	LoggerFrom(ctx).With("component", "test").Info("event")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["request_id"] != "request-123" || entry["trace_id"] != tid.String() {
		t.Fatal(entry)
	}
	if strings.Count(output.String(), `"request_id"`) != 1 || strings.Count(output.String(), `"trace_id"`) != 1 {
		t.Fatal("duplicate context keys")
	}
}

func TestRejectionsUseWarningAndFixedReason(t *testing.T) {
	for _, tc := range []struct{ file, reason string }{{"known-file-id", "virus"}, {"", "rejected_type"}} {
		var output bytes.Buffer
		LogRejection(context.Background(), NewLogger(&output, slog.LevelInfo), tc.file, tc.reason)
		var entry map[string]any
		if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
			t.Fatal(err)
		}
		if entry["level"] != "WARN" || entry["reason"] != tc.reason || entry["file_id"] == "" {
			t.Fatal(entry)
		}
	}
}
