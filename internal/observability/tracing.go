package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SetupTracing installs W3C propagation and a bounded batch exporter when enabled.
func SetupTracing(service string, logger *slog.Logger) (func(), error) {
	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", service))),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)))}
	switch exporter := os.Getenv("OTEL_TRACES_EXPORTER"); exporter {
	case "", "none":
	case "slog":
		options = append(options, sdktrace.WithBatcher(spanLogger{logger}, sdktrace.WithMaxQueueSize(256), sdktrace.WithMaxExportBatchSize(64)))
	default:
		return nil, fmt.Errorf("OTEL_TRACES_EXPORTER must be none or slog")
	}
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			logger.Error("trace shutdown failed", "error", err)
		}
	}, nil
}

type spanLogger struct{ logger *slog.Logger }

func (s spanLogger) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		s.logger.InfoContext(ctx, "trace span", "name", span.Name(), "trace_id", span.SpanContext().TraceID().String(),
			"span_id", span.SpanContext().SpanID().String(), "parent_span_id", span.Parent().SpanID().String(),
			"duration", span.EndTime().Sub(span.StartTime()), "status", span.Status().Code.String())
	}
	return nil
}
func (spanLogger) Shutdown(context.Context) error { return nil }
