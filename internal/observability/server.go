package observability

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func StartMetrics(address string, logger *slog.Logger, metrics *Metrics) (func(), error) {
	if metrics == nil {
		return nil, errors.New("metrics instance is required")
	}
	if address == "" {
		address = ":9090"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return serveMetrics(listener, logger, metrics), nil
}

func serveMetrics(listener net.Listener, logger *slog.Logger, metrics *Metrics) func() {
	mux := chi.NewRouter()
	mux.Use(metrics.HTTPMiddleware)
	mux.Use(RequestLogging(logger))
	mux.Get("/metrics", metrics.Handler().ServeHTTP)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server stopped", "error", err)
		}
	}()
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
		}
	}
}
