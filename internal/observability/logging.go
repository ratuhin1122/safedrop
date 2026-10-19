package observability

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewLoggerFromEnv(output io.Writer) (*slog.Logger, error) {
	level := slog.LevelInfo
	if value := strings.TrimSpace(os.Getenv("LOG_LEVEL")); value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
	}
	return NewLogger(output, level), nil
}

func RequestLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := ""
			if values := r.Header.Values("X-Request-ID"); len(values) == 1 && validRequestID(values[0]) {
				id = values[0]
			}
			if id == "" {
				var random [16]byte
				_, _ = rand.Read(random[:])
				id = hex.EncodeToString(random[:])
			}
			ctx := WithRequestID(r.Context(), id)
			ctx = withRequestLogger(ctx, logger)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", id)
			response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			started := time.Now()
			defer func() {
				status := response.Status()
				if status == 0 {
					status = http.StatusOK
				}
				route := "unmatched"
				if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
					route = rc.RoutePattern()
				}
				args := []any{"method", safeMethod(r.Method), "route", route, "status", status, "duration_ms", float64(time.Since(started)) / float64(time.Millisecond)}
				if user := UserID(ctx); user != "" {
					args = append(args, "user_id", user)
				}
				level := slog.LevelInfo
				if status >= 500 {
					level = slog.LevelError
				}
				LoggerFrom(ctx).Log(ctx, level, "http request", args...)
			}()
			next.ServeHTTP(response, r)
		})
	}
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func safeMethod(method string) string {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		return method
	}
	return "OTHER"
}
