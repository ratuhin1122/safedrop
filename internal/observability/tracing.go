package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func SetupTracing(fallbackService string, logger *slog.Logger) (func(), error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func() {}, nil
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("OTEL_EXPORTER_OTLP_ENDPOINT must be an HTTP(S) base URL without credentials or query")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(strings.TrimRight(endpoint, "/")+"/v1/traces"), otlptracehttp.WithTimeout(5*time.Second))
	if err != nil {
		return nil, errors.New("initialize OTLP HTTP trace exporter")
	}
	service := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if service == "" {
		service = fallbackService
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", service))),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
		sdktrace.WithBatcher(privateExporter{exporter}, sdktrace.WithMaxQueueSize(256), sdktrace.WithMaxExportBatchSize(64)))
	otel.SetTracerProvider(provider)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) { logger.Warn("trace export failed") }))
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			logger.Error("trace shutdown failed")
		}
	}, nil
}
