package jobs

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func captureUploadTraces(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	oldProvider, oldPropagation := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagation)
	})
	return exporter
}

func assertUploadTraces(t *testing.T, spans tracetest.SpanStubs) {
	t.Helper()
	groups := map[trace.TraceID]map[string]int{}
	parents := map[trace.SpanID]tracetest.SpanStub{}
	for _, span := range spans {
		id := span.SpanContext.TraceID()
		if groups[id] == nil {
			groups[id] = map[string]int{}
		}
		groups[id][span.Name]++
		parents[span.SpanContext.SpanID()] = span
	}
	if len(groups) != 2 {
		t.Fatalf("want two upload traces, got %d", len(groups))
	}
	for _, names := range groups {
		for _, name := range []string{"POST /v1/files", "validate.upload", "storage.put", "jobs.enqueue", "process_file", "scan.clamav"} {
			if names[name] != 1 {
				t.Errorf("want one %s per trace, got %d", name, names[name])
			}
		}
		if names["imageproc.convert"] == 1 && names["storage.move"] != 3 {
			t.Errorf("converted trace needs three storage moves: %v", names)
		}
	}
	converted := 0
	for _, span := range spans {
		if span.Name == "imageproc.convert" {
			converted++
		}
		if span.Name == "process_file" {
			parent := parents[span.Parent.SpanID()]
			if parent.Name != "jobs.enqueue" || parent.SpanContext.TraceID() != span.SpanContext.TraceID() || !span.Parent.IsRemote() {
				t.Error("worker did not restore its persisted enqueue parent")
			}
		}
	}
	if converted != 1 {
		t.Errorf("want one converted upload, got %d", converted)
	}
}
