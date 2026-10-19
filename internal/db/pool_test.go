package db

import (
	"context"
	"testing"
)

func TestOpenPoolRejectsInvalidURL(t *testing.T) {
	for _, dsn := range []string{"", " ", "://password-secret"} {
		pool, err := OpenPool(context.Background(), dsn)
		if err == nil || pool != nil {
			t.Fatalf("expected no pool and an error, got %v, %v", pool, err)
		}
	}
}

func TestOpenPoolCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pool, err := OpenPool(ctx, "postgres://localhost:1/safedrop?sslmode=disable")
	if err == nil || pool != nil {
		t.Fatalf("expected canceled connection to fail, got %v, %v", pool, err)
	}
}
