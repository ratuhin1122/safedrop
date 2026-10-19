package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/imageproc"
	"safedrop/internal/storage"
)

type failingOriginalDelete struct {
	storage.Store
	fail bool
}

func (s *failingOriginalDelete) Delete(ctx context.Context, key string) error {
	if s.fail && strings.HasPrefix(key, "incoming/") {
		return errors.New("temporary delete failure")
	}
	return s.Store.Delete(ctx, key)
}

func TestOriginalCleanupIsAfterCommitAndDurable(t *testing.T) {
	ctx, pool, file, _, _ := workerFixture(t)
	local, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	store := &failingOriginalDelete{Store: local, fail: true}
	if err := store.Put(ctx, file.StorageKey, strings.NewReader("source"), ""); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	for _, step := range [][2]string{{"uploaded", "scanning"}, {"scanning", "processing"}} {
		if _, err := q.AdvanceFile(ctx, db.AdvanceFileParams{ID: file.ID, ExpectedStatus: step[0], NextStatus: step[1]}); err != nil {
			t.Fatal(err)
		}
	}
	file.Status = "processing"
	engine, _ := imageproc.NewWithEncoder(imageproc.Options{MaxBytes: 1024, MaxPixels: 20000000}, &imageTestEncoder{})
	processor := &ImageProcessor{Engine: engine, Storage: store, Pool: pool}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if err := processor.Process(ctx, db.New(tx), file); err != nil {
		t.Fatal(err)
	}
	body, err := store.Get(ctx, file.StorageKey)
	if err != nil {
		t.Fatal("original deleted before commit")
	}
	body.Close()
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := processor.Finalize(ctx, file); err == nil {
		t.Fatal("delete failure ignored")
	}
	current, err := q.GetFile(ctx, file.ID)
	if err != nil || current.Status != "ready" {
		t.Fatalf("status=%s error=%v", current.Status, err)
	}
	if _, err := q.GetFileCleanup(ctx, file.ID); err != nil {
		t.Fatal("cleanup record lost")
	}
	store.fail = false
	if err := processor.Maintain(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if body, err := store.Get(ctx, file.StorageKey); !errors.Is(err, storage.ErrNotFound) {
		if body != nil {
			body.Close()
		}
		t.Fatalf("cleanup failed: %v", err)
	}
}
