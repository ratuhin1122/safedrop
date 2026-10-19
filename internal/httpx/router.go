package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"
	"safedrop/internal/observability"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterOption func(chi.Router)

func WithUploads(handler http.Handler) RouterOption {
	return func(r chi.Router) { r.Method(http.MethodPost, "/v1/files", handler) }
}

func NewRouter(logger *slog.Logger, trustedProxies []netip.Prefix, metrics *observability.Metrics, options ...RouterOption) http.Handler {
	r := chi.NewRouter()
	r.Use(TrustedProxy(trustedProxies))
	r.Use(metrics.HTTPMiddleware)
	r.Use(requestTelemetry)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				status := response.Status()
				if status == 0 {
					status = http.StatusOK
				}
				path := r.URL.Path
				if strings.Contains(path, "/s/") {
					path = "/s/[redacted]"
				}
				logger.InfoContext(r.Context(), "http request", "method", r.Method,
					"path", path, "client", r.RemoteAddr, "status", status,
					"duration", time.Since(started))
			}()
			next.ServeHTTP(response, r)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}
					logger.ErrorContext(r.Context(), "http panic", "panic", recovered)
					http.Error(w, "internal server error", http.StatusInternalServerError)
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
	return r
}
