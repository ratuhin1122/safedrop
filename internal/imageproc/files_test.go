package imageproc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"safedrop/internal/storage"
)

type failingMoveStore struct{ storage.Store }

func (failingMoveStore) Move(context.Context, string, string) error {
	return errors.New("move failed")
}

func TestUploadCleansStagingAfterMoveFailure(t *testing.T) {
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	path := filepath.Join(t.TempDir(), "encoded.webp")
	if err := os.WriteFile(path, []byte("encoded"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := upload(ctx, failingMoveStore{store}, "final/file/375.webp", path, "image/webp"); err == nil {
		t.Fatal("expected move failure")
	}
	for _, key := range []string{"incoming/processed/file/375.webp", "final/file/375.webp"} {
		body, err := store.Get(ctx, key)
		if err == nil {
			body.Close()
			t.Errorf("unexpected stored object %s", key)
		}
	}
}
