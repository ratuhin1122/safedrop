package observability

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingExportAndShutdown(t *testing.T) {
	oldProvider, oldPropagation := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	defer func() { otel.SetTracerProvider(oldProvider); otel.SetTextMapPropagator(oldPropagation) }()
	t.Setenv("OTEL_TRACES_EXPORTER", "slog")
	var logs bytes.Buffer
	shutdown, err := SetupTracing("test", NewLogger(&logs, slog.LevelInfo))
	if err != nil {
		t.Fatal(err)
	}
	tid, _ := trace.TraceIDFromHex("01234567890123456789012345678901")
	sid, _ := trace.SpanIDFromHex("0123456789012345")
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled, Remote: true,
	}))
	_, span := otel.Tracer("test").Start(ctx, "safe operation")
	span.End()
	shutdown()
	if !strings.Contains(logs.String(), `"msg":"trace span"`) || !strings.Contains(logs.String(), `"name":"safe operation"`) {
		t.Fatal(logs.String())
	}
	t.Setenv("OTEL_TRACES_EXPORTER", "invalid")
	if _, err := SetupTracing("test", NewLogger(&logs, slog.LevelInfo)); err == nil {
		t.Fatal("accepted invalid exporter")
	}
}
