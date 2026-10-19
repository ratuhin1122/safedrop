package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/auth"
	"safedrop/internal/config"
	"safedrop/internal/db"
	"safedrop/internal/httpx"
	"safedrop/internal/jobs"
	"safedrop/internal/share"
	"safedrop/internal/storage"
)

func setupRoutes(ctx context.Context, cfg config.Upload, logger *slog.Logger) ([]httpx.RouterOption, func(), error) {
	if cfg.DatabaseURL == "" {
		unavailable := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "DATABASE_URL is required", http.StatusServiceUnavailable)
		})
		return []httpx.RouterOption{func(r chi.Router) {
			r.Method("POST", "/v1/files", unavailable)
			r.Method("POST", "/v1/files/{id}/shares", unavailable)
			r.Method("GET", "/s/{token}", unavailable)
		}}, func() {}, nil
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
	shares := &httpx.ShareHandler{Auth: authenticator, Logger: logger,
		Service: &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store, Presign: cfg.SharePresign}}
	return []httpx.RouterOption{httpx.WithUploads(handler), httpx.WithShares(shares)}, func() { pool.Close(); closeStore() }, nil
}
