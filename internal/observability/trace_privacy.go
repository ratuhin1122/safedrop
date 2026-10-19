package observability

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// otelhttp captures URL attributes by default. Export only these safe fields;
// never send raw paths, query strings, headers, filenames, bodies, or error text.
type privateExporter struct{ sdktrace.SpanExporter }

func (e privateExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	filtered := make([]sdktrace.ReadOnlySpan, len(spans))
	for i, span := range spans {
		filtered[i] = privateSpan{span}
	}
	return e.SpanExporter.ExportSpans(ctx, filtered)
}

type privateSpan struct{ sdktrace.ReadOnlySpan }

func (s privateSpan) Attributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0)
	for _, attr := range s.ReadOnlySpan.Attributes() {
		switch string(attr.Key) {
		case "http.request.method", "http.response.status_code", "http.route", "job.id", "job.attempt":
			attrs = append(attrs, attr)
		}
	}
	return attrs
}
