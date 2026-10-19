package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"safedrop/internal/observability"
)

func requestTelemetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bytes [16]byte
		_, _ = rand.Read(bytes[:])
		id := hex.EncodeToString(bytes[:])
		ctx := observability.WithRequestID(r.Context(), id)
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("safedrop/http").Start(ctx, "HTTP request", trace.WithSpanKind(trace.SpanKindServer))
		started := time.Now()
		defer span.End()
		w.Header().Set("X-Request-ID", id)
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		r = r.WithContext(ctx)
		defer func() {
			status := response.Status()
			if status == 0 {
				status = 200
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				matched := chi.NewRouteContext()
				if routes := chi.RouteContext(r.Context()).Routes; routes != nil && routes.Match(matched, r.Method, r.URL.Path) {
					route = matched.RoutePattern()
				}
			}
			if status == http.StatusMethodNotAllowed || route == "" {
				route = "unmatched"
			}
			method := metricMethod(r.Method)
			observability.DefaultMetrics.HTTPRequests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
			observability.DefaultMetrics.HTTPDuration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
			span.SetName("HTTP " + strconv.Itoa(status))
			if route != "" {
				span.SetName(r.Method + " " + route)
			}
			span.SetAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route), attribute.Int("http.response.status_code", status))
			if status >= 500 {
				span.SetStatus(codes.Error, "HTTP server error")
			}
		}()
		next.ServeHTTP(response, r)
	})
}

func WithMetrics(depth observability.QueueStates) RouterOption {
	return func(r chi.Router) { r.Handle("/metrics", observability.DefaultMetrics.Handler(depth)) }
}

func metricMethod(method string) string {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		return method
	}
	return "OTHER"
}
