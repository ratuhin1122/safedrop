package share_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/share"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func databaseFixture(t *testing.T) (context.Context, *pgxpool.Pool, *storage.Local, db.File, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	pool, err := db.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	q := db.New(pool)
	user, err := q.CreateUser(ctx, db.CreateUserParams{Email: rand.Text() + "@share.test", PasswordHash: "test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanup, "DELETE FROM files WHERE owner_id=$1", user.ID)
		_, _ = pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", user.ID)
	})
	incoming := "incoming/" + rand.Text()
	file, err := q.CreateFile(ctx, db.CreateFileParams{OwnerID: user.ID, OriginalName: "image.jpg", Mime: "image/jpeg", Size: 3, StorageKey: incoming})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, incoming, strings.NewReader("jpg"), "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	for _, step := range [][2]string{{"uploaded", "scanning"}, {"scanning", "processing"}} {
		if _, err := q.AdvanceFile(ctx, db.AdvanceFileParams{ID: file.ID, ExpectedStatus: step[0], NextStatus: step[1]}); err != nil {
			t.Fatal(err)
		}
	}
	for _, width := range []int32{375, 768, 1200} {
		key := fmt.Sprintf("final/%s/%d.webp", file.ID.String(), width)
		if err := store.Put(ctx, key, strings.NewReader("img"), "image/webp"); err != nil {
			t.Fatal(err)
		}
		if err := q.UpsertFileVariant(ctx, db.UpsertFileVariantParams{FileID: file.ID, TargetWidth: width, Width: width, Height: 200, Size: 3, Mime: "image/webp", StorageKey: key}); err != nil {
			t.Fatal(err)
		}
		file.StorageKey = key
	}
	if _, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: file.ID, StorageKey: file.StorageKey, Mime: "image/webp", Size: 3, Width: pgtype.Int4{Int32: 1200, Valid: true}, Height: pgtype.Int4{Int32: 200, Valid: true}}); err != nil {
		t.Fatal(err)
	}
	if err := q.ScheduleFileCleanup(ctx, db.ScheduleFileCleanupParams{FileID: file.ID, StorageKey: incoming}); err != nil {
		t.Fatal(err)
	}
	file, err = q.GetFile(ctx, file.ID)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool, store, file, incoming
}

func newLink(t *testing.T, ctx context.Context, service *share.Service, file db.File, max *int64) string {
	t.Helper()
	created, err := service.Create(ctx, file.OwnerID, file.ID, share.CreateOptions{ExpiresAt: time.Now().Add(time.Hour), MaxDownloads: max})
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimPrefix(created.URL, "/s/")
}

func expireLink(t *testing.T, ctx context.Context, pool *pgxpool.Pool, token string) {
	t.Helper()
	raw, _ := base64.RawURLEncoding.DecodeString(token)
	hash := sha256.Sum256(raw)
	if _, err := pool.Exec(ctx, "UPDATE share_links SET expires_at=now()-interval '1 minute' WHERE token_hash=$1", hash[:]); err != nil {
		t.Fatal(err)
	}
}
