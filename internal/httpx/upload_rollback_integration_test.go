package httpx

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
)

type failAfterEnqueue struct {
	queue  *jobs.Queue
	fileID string
}

func (q *failAfterEnqueue) Enqueue(ctx context.Context, tx pgx.Tx, id string) error {
	q.fileID = id
	if err := q.queue.Enqueue(ctx, tx, id); err != nil {
		return err
	}
	return errors.New("forced failure after River insert")
}

func TestUploadRollsBackFileAndRiverJob(t *testing.T) {
	h, _, pool, user := integrationUpload(t)
	queue, err := jobs.NewQueue(pool, h.Logger)
	if err != nil {
		t.Fatal(err)
	}
	failing := &failAfterEnqueue{queue: queue}
	repository := &UploadDatabase{Pool: pool, Queue: failing}
	var key string
	h.Repository = uploadRepositoryFunc(func(ctx context.Context, p db.CreateFileParams) (db.File, error) {
		key = p.StorageKey
		return repository.CreateUpload(ctx, p)
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, uploadRequest(uploadPNG(t)))
	if w.Code != 500 || failing.fileID == "" {
		t.Fatalf("response=%d job file=%s", w.Code, failing.fileID)
	}
	ctx := context.Background()
	var files, queued int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM files WHERE owner_id=$1", user.ID).Scan(&files); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE args->>'file_id'=$1", failing.fileID).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if files != 0 || queued != 0 {
		t.Fatalf("transaction leaked files=%d jobs=%d", files, queued)
	}
	if body, err := h.Storage.Get(ctx, key); !errors.Is(err, storage.ErrNotFound) {
		if body != nil {
			body.Close()
		}
		t.Fatalf("incoming object leaked: %v", err)
	}
}
