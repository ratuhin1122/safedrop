package share_test

import (
	"errors"
	"fmt"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/share"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
)

func TestExpiryPreservesActiveLinksThenDeletesAllFiles(t *testing.T) {
	ctx, pool, store, file, incoming := databaseFixture(t)
	service := &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store}
	first := newLink(t, ctx, service, file, nil)
	active := newLink(t, ctx, service, file, nil)
	expireLink(t, ctx, pool, first)
	cleaner := &share.Cleaner{Pool: pool, Storage: store}
	if err := cleaner.Clean(ctx); err != nil {
		t.Fatal(err)
	}
	download, err := service.Open(ctx, active, "")
	if err != nil {
		t.Fatal("active link lost its file")
	}
	download.Body.Close()
	if _, err := service.Open(ctx, first, ""); !errors.Is(err, share.ErrGone) {
		t.Fatalf("expired link: %v", err)
	}
	expireLink(t, ctx, pool, active)
	if err := cleaner.Clean(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.New(pool).GetFile(ctx, file.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("file row remains: %v", err)
	}
	keys := []string{incoming}
	for _, width := range []int{375, 768, 1200} {
		keys = append(keys, fmt.Sprintf("final/%s/%d.webp", file.ID.String(), width))
	}
	for _, key := range keys {
		if body, err := store.Get(ctx, key); !errors.Is(err, storage.ErrNotFound) {
			if body != nil {
				body.Close()
			}
			t.Fatalf("object remains %s: %v", key, err)
		}
	}
	if err := cleaner.Clean(ctx); err != nil {
		t.Fatalf("cleanup replay: %v", err)
	}
}
