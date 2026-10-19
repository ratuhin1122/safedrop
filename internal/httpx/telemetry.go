package httpx

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func requestTelemetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
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
			span.SetName("HTTP " + strconv.Itoa(status))
			method := r.Method
			switch method {
			case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
			default:
				method = "OTHER"
			}
			if route != "" {
				span.SetName(method + " " + route)
			}
			span.SetAttributes(attribute.String("http.request.method", method), attribute.String("http.route", route), attribute.Int("http.response.status_code", status))
			if status >= 500 {
				span.SetStatus(codes.Error, "HTTP server error")
			}
		}()
		next.ServeHTTP(response, r)
	})
}
