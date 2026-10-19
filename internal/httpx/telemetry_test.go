package httpx

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"safedrop/internal/observability"
)

func TestRequestTelemetry(t *testing.T) {
	previous, propagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(previous)
		otel.SetTextMapPropagator(propagator)
	}()
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelInfo).With("service", "test")
	router := NewRouter(logger, nil, func(r chi.Router) {
		r.Get("/s/{token}", func(w http.ResponseWriter, r *http.Request) {
			logger.InfoContext(r.Context(), "handler ran")
			if observability.RequestID(r.Context()) == "" {
				t.Error("missing context request ID")
			}
			w.WriteHeader(200)
		})
	})
	r := httptest.NewRequest("GET", "/s/secret-token", nil)
	r.Header.Set("X-Request-ID", "untrusted-id")
	r.Header.Set("traceparent", "00-01234567890123456789012345678901-0123456789012345-01")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	id := w.Header().Get("X-Request-ID")
	if len(id) != 32 || !strings.Contains(logs.String(), `"request_id":"`+id+`"`) {
		t.Fatal(logs.String())
	}
	if strings.Contains(logs.String(), "secret-token") {
		t.Fatal("token leaked in log")
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Name != "GET /s/{token}" || spans[0].SpanContext.TraceID().String() != "01234567890123456789012345678901" {
		t.Fatalf("spans: %+v", spans)
	}
	if !strings.Contains(logs.String(), `"trace_id":"01234567890123456789012345678901"`) {
		t.Fatal(logs.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	h, _ := testUploadHandler(t)
	router := NewRouter(h.Logger, nil, WithMetrics(func(context.Context) (map[string]int64, error) { return map[string]int64{"available": 12}, nil }))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `safedrop_queue_jobs{state="available"} 12`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	NewRouter(h.Logger, nil).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
