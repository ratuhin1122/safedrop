package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/observability"
)

func main() {
	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	if err := migrate(logger); err != nil {
		logger.Error("River migration failed", "error", err)
		os.Exit(1)
	}
}

func migrate(logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := db.OpenPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	return jobs.Migrate(ctx, pool, logger)
}
