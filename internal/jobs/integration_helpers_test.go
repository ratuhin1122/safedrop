package jobs

import (
	"bytes"
	"context"
	"crypto/rand"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type lockedLog struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (l *lockedLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.Write(p)
}
func (l *lockedLog) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.buffer.String() }

func workerFixture(t *testing.T) (context.Context, *pgxpool.Pool, db.File, *slog.Logger, *lockedLog) {
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
	logs := &lockedLog{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	if err := Migrate(ctx, pool, logger); err != nil {
		t.Fatal(err)
	}
	user, err := db.New(pool).CreateUser(ctx, db.CreateUserParams{Email: rand.Text() + "@worker.test", PasswordHash: "test-hash"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanup, `DELETE FROM river_job WHERE args->>'file_id' IN (SELECT id::text FROM files WHERE owner_id=$1)`, user.ID)
		_, _ = pool.Exec(cleanup, "DELETE FROM files WHERE owner_id=$1", user.ID)
		_, _ = pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", user.ID)
	})
	file, err := db.New(pool).CreateFile(ctx, db.CreateFileParams{OwnerID: user.ID,
		OriginalName: "image.png", Mime: "image/png", Size: 1, StorageKey: "incoming/" + rand.Text()})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool, file, logger, logs
}

func testJob(file db.File, attempt, maxAttempts int) *river.Job[ProcessFileArgs] {
	return &river.Job[ProcessFileArgs]{Args: ProcessFileArgs{FileID: file.ID.String()},
		JobRow: &rivertype.JobRow{ID: 1, Attempt: attempt, MaxAttempts: maxAttempts}}
}

func readyProcessor(ctx context.Context, q *db.Queries, file db.File) error {
	if _, err := q.AdvanceFile(ctx, db.AdvanceFileParams{ID: file.ID, ExpectedStatus: "scanning", NextStatus: "processing"}); err != nil {
		return err
	}
	_, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: file.ID, StorageKey: "final/" + file.ID.String(), Mime: file.Mime, Size: file.Size})
	return err
}
