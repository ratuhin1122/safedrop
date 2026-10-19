package observability

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func restoreTracing(t *testing.T) {
	t.Helper()
	provider, propagation, handler := otel.GetTracerProvider(), otel.GetTextMapPropagator(), otel.GetErrorHandler()
	t.Cleanup(func() {
		otel.SetTracerProvider(provider)
		otel.SetTextMapPropagator(propagation)
		otel.SetErrorHandler(handler)
	})
}

func TestTracingWithoutEndpointIsNoop(t *testing.T) {
	restoreTracing(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	shutdown, err := SetupTracing("test", NewLogger(io.Discard, slog.LevelInfo))
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()
	_, span := otel.Tracer("test").Start(context.Background(), "disabled")
	defer span.End()
	if span.IsRecording() || span.SpanContext().IsValid() {
		t.Fatal("expected no-op root span")
	}
}

func TestOTLPHTTPExportsAndFlushesWithoutPrivateAttributes(t *testing.T) {
	restoreTracing(t)
	requests := make(chan *collector.ExportTraceServiceRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" || r.Method != "POST" {
			t.Error("invalid OTLP endpoint")
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if bytes.Contains(payload, []byte("private-secret")) {
			t.Error("private trace attribute exported")
		}
		request := &collector.ExportTraceServiceRequest{}
		if err := proto.Unmarshal(payload, request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, _ = w.Write([]byte{})
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_SERVICE_NAME", "custom-service")
	shutdown, err := SetupTracing("fallback", NewLogger(io.Discard, slog.LevelInfo))
	if err != nil {
		t.Fatal(err)
	}
	_, span := otel.Tracer("test").Start(context.Background(), "safe operation")
	span.SetAttributes(attribute.String("url.full", "private-secret"), attribute.String("http.route", "/s/{token}"))
	span.End()
	shutdown()
	select {
	case request := <-requests:
		if len(request.ResourceSpans) != 1 {
			t.Fatal("missing resource")
		}
		resource := request.ResourceSpans[0]
		found := false
		for _, attr := range resource.Resource.Attributes {
			if attr.Key == "service.name" && attr.Value.GetStringValue() == "custom-service" {
				found = true
			}
		}
		if !found || len(resource.ScopeSpans) != 1 || len(resource.ScopeSpans[0].Spans) != 1 {
			t.Fatal("missing configured service/span")
		}
	default:
		t.Fatal("shutdown did not flush OTLP spans")
	}
}

func TestInvalidOTLPEndpointFails(t *testing.T) {
	restoreTracing(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "not-a-url")
	if _, err := SetupTracing("test", NewLogger(io.Discard, slog.LevelInfo)); err == nil {
		t.Fatal("accepted invalid endpoint")
	}
}
