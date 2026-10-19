package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Run against a migrated test database with TEST_DATABASE_URL set.
func testQueries(t *testing.T) (context.Context, *Queries) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return ctx, New(tx)
}

func TestFileAndShareLifecycle(t *testing.T) {
	ctx, q := testQueries(t)
	u, err := q.CreateUser(ctx, CreateUserParams{Email: "flow@example.test", PasswordHash: "hash"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := q.CreateFile(ctx, CreateFileParams{OwnerID: u.ID, OriginalName: "image.png",
		Mime: "image/png", Size: 42, StorageKey: "incoming/test"})
	if err != nil {
		t.Fatal(err)
	}
	n, err := q.AdvanceFile(ctx, AdvanceFileParams{ID: f.ID, ExpectedStatus: "uploaded", NextStatus: "processing"})
	if err != nil || n != 0 {
		t.Fatalf("skipped scan: %d %v", n, err)
	}
	share := CreateShareLinkParams{TokenHash: make([]byte, 32), FileID: f.ID, OwnerID: u.ID,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		MaxDownloads: pgtype.Int8{Int64: 1, Valid: true}}
	if _, err := q.CreateShareLink(ctx, share); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("shared unready file: %v", err)
	}
	for _, step := range [][2]string{{"uploaded", "scanning"}, {"scanning", "processing"}} {
		n, err = q.AdvanceFile(ctx, AdvanceFileParams{ID: f.ID, ExpectedStatus: step[0], NextStatus: step[1]})
		if err != nil || n != 1 {
			t.Fatalf("transition: %d %v", n, err)
		}
	}
	n, err = q.CompleteFile(ctx, CompleteFileParams{ID: f.ID, StorageKey: "final/test", Mime: "image/png", Size: 40})
	if err != nil || n != 1 {
		t.Fatalf("complete: %d %v", n, err)
	}
	if _, err = q.CreateShareLink(ctx, share); err != nil {
		t.Fatal(err)
	}
	s, err := q.ConsumeShareDownload(ctx, share.TokenHash)
	if err != nil || s.DownloadCount != 1 {
		t.Fatalf("download: %+v %v", s, err)
	}
	if _, err = q.ConsumeShareDownload(ctx, share.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("exceeded download limit: %v", err)
	}
	share.TokenHash[0] = 1
	share.MaxDownloads = pgtype.Int8{}
	share.ExpiresAt.Time = time.Now().Add(-time.Hour)
	if _, err = q.CreateShareLink(ctx, share); err != nil {
		t.Fatal(err)
	}
	if _, err = q.ConsumeShareDownload(ctx, share.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("downloaded expired link: %v", err)
	}
	share.TokenHash[0] = 2
	share.ExpiresAt.Time = time.Now().Add(time.Hour)
	if _, err = q.CreateShareLink(ctx, share); err != nil {
		t.Fatal(err)
	}
	n, err = q.SoftDeleteFile(ctx, SoftDeleteFileParams{ID: f.ID, OwnerID: u.ID})
	if err != nil || n != 1 {
		t.Fatalf("delete: %d %v", n, err)
	}
	if _, err = q.GetFile(ctx, f.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("deleted file visible: %v", err)
	}
	if _, err = q.ConsumeShareDownload(ctx, share.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("downloaded deleted file: %v", err)
	}
}
