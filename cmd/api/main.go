package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"safedrop/internal/config"
	"safedrop/internal/httpx"
	"safedrop/internal/observability"
)

func main() {
	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	if err := run(); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := observability.NewLogger(os.Stdout, cfg.LogLevel).With("service", "safedrop-api")
	slog.SetDefault(logger)
	closeTracing, err := observability.SetupTracing("safedrop-api", logger)
	if err != nil {
		return err
	}
	defer closeTracing()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	routes, queueMetrics, closeUploads, err := setupRoutes(ctx, cfg.Upload, logger)
	if err != nil {
		return err
	}
	defer closeUploads()
	closeMetrics, err := observability.StartMetrics(os.Getenv("METRICS_ADDR"), queueMetrics, logger)
	if err != nil {
		return err
	}
	defer closeMetrics()
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: httpx.NewRouter(logger, cfg.TrustedProxies, routes...),
		ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 1 << 20, ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	logger.Info("api starting", "address", cfg.HTTPAddr)
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		stop()
	}
	logger.Info("api shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("api stopped")
	return nil
}
