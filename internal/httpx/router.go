package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"
	"safedrop/internal/observability"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RouterOption func(chi.Router)

func WithUploads(handler http.Handler) RouterOption {
	return func(r chi.Router) { r.Method(http.MethodPost, "/v1/files", handler) }
}

func NewRouter(logger *slog.Logger, trustedProxies []netip.Prefix, metrics *observability.Metrics, options ...RouterOption) http.Handler {
	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { Error(w, "not found", 404) })
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) { Error(w, "method not allowed", 405) })
	r.Use(TrustedProxy(trustedProxies))
	r.Use(metrics.HTTPMiddleware)
	r.Use(requestTelemetry)
	r.Use(observability.RequestLogging(logger))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}
					Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})
	for _, option := range options {
		option(r)
	}
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	return otelhttp.NewHandler(r, "HTTP request", otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string { return "HTTP request" }))
}
