package httpx

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

func integrationUpload(t *testing.T) (*UploadHandler, *storage.Local, *pgxpool.Pool, db.User) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := db.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h, store := testUploadHandler(t)
	if err := jobs.Migrate(ctx, pool, h.Logger); err != nil {
		t.Fatal(err)
	}
	user, err := db.New(pool).CreateUser(ctx, db.CreateUserParams{Email: rand.Text() + "@upload.test", PasswordHash: "unused-stub-hash"})
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
	h.Auth, err = auth.NewStub(user.ID.String(), uploadTestToken)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := jobs.NewQueue(pool, h.Logger)
	if err != nil {
		t.Fatal(err)
	}
	h.Repository = &UploadDatabase{Pool: pool, Queue: queue}
	return h, store, pool, user
}
