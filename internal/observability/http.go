package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// HTTPMiddleware reads the final chi pattern after routing, never the raw path.
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			status := response.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := "unmatched"
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				route = rc.RoutePattern()
			}
			method := metricMethod(r.Method)
			m.HTTPRequests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
			m.HTTPDuration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
		}()
		next.ServeHTTP(response, r)
	})
}

func metricMethod(method string) string {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		return method
	}
	return "OTHER"
}
