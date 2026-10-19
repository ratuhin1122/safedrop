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

func StartMetrics(address string, depth QueueStates, logger *slog.Logger) (func(), error) {
	if address == "" {
		address = ":9090"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return serveMetrics(listener, depth, logger), nil
}

func serveMetrics(listener net.Listener, depth QueueStates, logger *slog.Logger) func() {
	mux := chi.NewRouter()
	mux.Use(DefaultMetrics.HTTPMiddleware)
	mux.Get("/metrics", DefaultMetrics.Handler(depth).ServeHTTP)
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
