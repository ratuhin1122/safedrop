package main

import (
	"context"
	"log/slog"

	"safedrop/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func migrateApplication(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	database := stdlib.OpenDBFromPool(pool)
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.Files)
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	logger.Info("application migrations complete", "applied", len(results))
	return nil
}
