package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"safedrop/internal/config"
	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/observability"
	"safedrop/internal/share"
)

func main() {
	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	if err := run(); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadWorker()
	if err != nil {
		return err
	}
	logger := observability.NewLogger(os.Stdout, cfg.LogLevel).With("service", "safedrop-worker")
	slog.SetDefault(logger)
	closeTracing, err := observability.SetupTracing("safedrop-worker", logger)
	if err != nil {
		return err
	}
	defer closeTracing()
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := db.OpenPool(signals, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	closeMetrics, err := observability.StartMetrics(os.Getenv("METRICS_ADDR"), jobs.QueueMetrics(pool), logger)
	if err != nil {
		return err
	}
	defer closeMetrics()
	processor, closeStorage, err := setupScanner(cfg.Scan, pool, logger)
	if err != nil {
		return err
	}
	defer closeStorage()
	client, err := jobs.NewWorkerClient(pool, logger, jobs.WorkerOptions{
		Concurrency: cfg.Concurrency, JobTimeout: cfg.JobTimeout,
		ShareCleaner: &share.Cleaner{Pool: pool, Storage: processor.Storage},
	}, processor)
	if err != nil {
		return err
	}
	startup, cancelStartup := context.WithTimeout(signals, cfg.ShutdownTimeout)
	_, err = jobs.ReconcileExhausted(startup, pool)
	cancelStartup()
	if err != nil {
		return err
	}
	workCtx, cancelWork := context.WithCancel(context.Background())
	defer cancelWork()
	if err := client.Start(workCtx); err != nil {
		return err
	}
	maintenance, cancelMaintenance := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); jobs.RunReconciler(maintenance, pool, logger, processor) }()
	defer func() { cancelMaintenance(); <-done }()
	logger.Info("worker started", "concurrency", cfg.Concurrency)
	select {
	case <-signals.Done():
		stop()
	case <-client.Stopped():
		return errors.New("River client stopped unexpectedly")
	}
	logger.Info("worker shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := client.Stop(shutdown); err != nil {
		force, cancelForce := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelForce()
		return errors.Join(err, client.StopAndCancel(force))
	}
	logger.Info("worker stopped")
	return nil
}
