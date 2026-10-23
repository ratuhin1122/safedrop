package main

import (
	"testing"

	"safedrop/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestEmbeddedApplicationMigrations(t *testing.T) {
	config, err := pgx.ParseConfig("postgres://localhost/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	database := stdlib.OpenDB(*config)
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	sources := provider.ListSources()
	if len(sources) != 5 {
		t.Fatalf("embedded migrations = %d, want 5", len(sources))
	}
	for index, source := range sources {
		if source.Version != int64(index+1) {
			t.Fatalf("migration %d has version %d", index, source.Version)
		}
	}
}
