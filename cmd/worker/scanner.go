package main

import (
	"log/slog"

	"safedrop/internal/config"
	"safedrop/internal/jobs"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

func setupScanner(cfg config.WorkerScan, logger *slog.Logger) (*jobs.ScanProcessor, func(), error) {
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
	return &jobs.ScanProcessor{Scanner: scanner, Storage: store, Logger: logger}, closeStore, nil
}
