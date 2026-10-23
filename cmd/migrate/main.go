package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/observability"
)

func main() {
	application := flag.Bool("all", false, "apply application migrations before River migrations")
	flag.Parse()
	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	if err := migrate(logger, *application); err != nil {
		logger.Error("database migrations failed", "error", err)
		os.Exit(1)
	}
}

func migrate(logger *slog.Logger, application bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pool, err := db.OpenPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if application {
		if err := migrateApplication(ctx, pool, logger); err != nil {
			return err
		}
	}
	return jobs.Migrate(ctx, pool, logger)
}
