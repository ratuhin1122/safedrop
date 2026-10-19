package jobs

import (
	"testing"
	"time"

	"safedrop/internal/db"

	"github.com/riverqueue/river"
)

func TestReconcileDiscardedJob(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	client, err := NewWorkerClient(pool, logger, WorkerOptions{Concurrency: 1, JobTimeout: time.Minute}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inserted, err := client.Insert(ctx, ProcessFileArgs{FileID: file.ID.String()}, &river.InsertOpts{MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, "UPDATE river_job SET state='discarded', attempt=max_attempts, finalized_at=now() WHERE id=$1", inserted.Job.ID)
	if err != nil {
		t.Fatal(err)
	}
	count, err := ReconcileExhausted(ctx, pool)
	if err != nil || count != 1 {
		t.Fatalf("reconciled=%d error=%v", count, err)
	}
	current, err := db.New(pool).GetFile(ctx, file.ID)
	if err != nil || current.Status != "rejected" {
		t.Fatalf("file=%+v error=%v", current, err)
	}
	if count, err := ReconcileExhausted(ctx, pool); err != nil || count != 0 {
		t.Fatalf("not idempotent: %d %v", count, err)
	}
}
