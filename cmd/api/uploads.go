package main

import (
	"context"
	"log/slog"
	"net/http"

	"safedrop/internal/auth"
	"safedrop/internal/config"
	"safedrop/internal/db"
	"safedrop/internal/httpx"
	"safedrop/internal/jobs"
	"safedrop/internal/storage"
)

func setupUploads(ctx context.Context, cfg config.Upload, logger *slog.Logger) (http.Handler, func(), error) {
	if cfg.DatabaseURL == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "uploads require DATABASE_URL", http.StatusServiceUnavailable)
		}), func() {}, nil
	}
	authenticator, err := auth.NewStub(cfg.StubUserID, cfg.StubToken)
	if err != nil {
		return nil, nil, err
	}
	pool, err := db.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	queue, err := jobs.NewQueue(pool, logger)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	var store storage.Store
	closeStore := func() {}
	if cfg.Backend == "local" {
		local, err := storage.NewLocal(storage.LocalConfig{Root: cfg.LocalRoot})
		if err != nil {
			pool.Close()
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
			pool.Close()
			return nil, nil, err
		}
	}
	handler := &httpx.UploadHandler{Auth: authenticator, Storage: store,
		Repository: &httpx.UploadDatabase{Pool: pool, Queue: queue}, Limits: cfg.Limits, Logger: logger}
	return handler, func() { pool.Close(); closeStore() }, nil
}
