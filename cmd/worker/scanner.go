package main

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"safedrop/internal/config"
	"safedrop/internal/imageproc"
	"safedrop/internal/jobs"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

func setupScanner(cfg config.WorkerScan, pool *pgxpool.Pool, logger *slog.Logger) (*jobs.ScanProcessor, func(), error) {
	engine, err := imageproc.New(imageproc.Options{AVIF: cfg.AVIF, MaxBytes: cfg.MaxBytes, MaxPixels: cfg.MaxPixels, TempDir: cfg.TempDir})
	if err != nil {
		return nil, nil, err
	}
	scanner, err := scan.NewClient(cfg.Address, cfg.Timeout, cfg.MaxBytes)
	if err != nil {
		return nil, nil, err
	}
	var store storage.Store
	closeStore := func() {}
	if cfg.Backend == "local" {
		local, err := storage.NewLocal(storage.LocalConfig{Root: cfg.LocalRoot})
		if err != nil {
			return nil, nil, err
		}
		store = local
		closeStore = func() {
			if err := local.Close(); err != nil {
				logger.Error("storage close failed", "error", err)
			}
		}
	} else {
		store, err = storage.NewS3(cfg.S3)
		if err != nil {
			return nil, nil, err
		}
	}
	processor := &jobs.ImageProcessor{Engine: engine, Storage: store, Pool: pool}
	return &jobs.ScanProcessor{Scanner: scanner, Storage: store, Logger: logger, Next: processor}, closeStore, nil
}
