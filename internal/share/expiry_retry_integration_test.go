package share_test

import (
	"context"
	"errors"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/share"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
)

type failingDelete struct {
	storage.Store
	fail bool
}

func (s *failingDelete) Delete(ctx context.Context, key string) error {
	if s.fail {
		return errors.New("storage offline")
	}
	return s.Store.Delete(ctx, key)
}

func TestExpiryDeletionRetriesAfterStorageFailure(t *testing.T) {
	ctx, pool, store, file, _ := databaseFixture(t)
	service := &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store}
	token := newLink(t, ctx, service, file, nil)
	expireLink(t, ctx, pool, token)
	failing := &failingDelete{Store: store, fail: true}
	cleaner := &share.Cleaner{Pool: pool, Storage: failing}
	if err := cleaner.Clean(ctx); err == nil {
		t.Fatal("deletion error ignored")
	}
	if _, err := service.Open(ctx, token, ""); !errors.Is(err, share.ErrGone) {
		t.Fatalf("expired link was accessible: %v", err)
	}
	if _, err := db.New(pool).GetExpiredFile(ctx, file.ID); err != nil {
		t.Fatalf("pending deletion record lost: %v", err)
	}
	failing.fail = false
	if err := cleaner.Clean(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.New(pool).GetExpiredFile(ctx, file.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("file not removed: %v", err)
	}
}
