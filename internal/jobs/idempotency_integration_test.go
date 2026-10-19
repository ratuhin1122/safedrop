package jobs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"safedrop/internal/db"
)

func TestConcurrentDuplicatesProcessOnce(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	var calls atomic.Int32
	worker := &ProcessFile{Pool: pool, Logger: logger, Processor: processorFunc(func(ctx context.Context, q *db.Queries, f db.File) error {
		calls.Add(1)
		return readyProcessor(ctx, q, f)
	})}
	errs := make(chan error, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() { defer group.Done(); errs <- worker.Work(ctx, testJob(file, 1, 5)) }()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("processor calls=%d", calls.Load())
	}
	current, err := db.New(pool).GetFile(ctx, file.ID)
	if err != nil || current.Status != "ready" {
		t.Fatalf("file=%+v error=%v", current, err)
	}
}

func TestRejectedAndDeletedFilesAreSkipped(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	worker := &ProcessFile{Pool: pool, Logger: logger}
	if err := worker.Work(ctx, testJob(file, 1, 1)); !errors.Is(err, ErrPipelineUnavailable) {
		t.Fatalf("expected pipeline failure: %v", err)
	}
	worker.Processor = processorFunc(func(context.Context, *db.Queries, db.File) error {
		t.Error("processed terminal/deleted file")
		return nil
	})
	if err := worker.Work(ctx, testJob(file, 1, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.New(pool).SoftDeleteFile(ctx, db.SoftDeleteFileParams{ID: file.ID, OwnerID: file.OwnerID}); err != nil {
		t.Fatal(err)
	}
	if err := worker.Work(ctx, testJob(file, 1, 5)); err != nil {
		t.Fatal(err)
	}
}
