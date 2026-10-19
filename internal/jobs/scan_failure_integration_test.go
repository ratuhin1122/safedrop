package jobs

import (
	"context"
	"errors"
	"io"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/scan"
)

func TestScanFailureDoesNotAdvanceFile(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	store := &scanStore{data: []byte("image")}
	pipeline := &ScanProcessor{Storage: store, Logger: logger,
		Scanner: scannerFunc(func(context.Context, io.Reader) (scan.Result, error) {
			return scan.Result{}, errors.New("clamd offline")
		})}
	worker := &ProcessFile{Pool: pool, Logger: logger, Processor: pipeline}
	if err := worker.Work(ctx, testJob(file, 1, 5)); err == nil {
		t.Fatal("scanner failure passed")
	}
	current, err := db.New(pool).GetFile(ctx, file.ID)
	if err != nil || current.Status != "scanning" || store.deleted {
		t.Fatalf("status=%s deleted=%v error=%v", current.Status, store.deleted, err)
	}
}

func TestInfectedCleanupRetriesWithoutRescanning(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	store := &scanStore{data: []byte("infected"), failDeletes: 1}
	calls := 0
	pipeline := &ScanProcessor{Storage: store, Logger: logger,
		Scanner: scannerFunc(func(context.Context, io.Reader) (scan.Result, error) {
			calls++
			return scan.Result{Verdict: scan.Infected, Signature: "test"}, nil
		})}
	worker := &ProcessFile{Pool: pool, Logger: logger, Processor: pipeline}
	if err := worker.Work(ctx, testJob(file, 1, 5)); err == nil {
		t.Fatal("cleanup failure was acknowledged")
	}
	current, err := db.New(pool).GetFile(ctx, file.ID)
	if err != nil || current.Status != "rejected" {
		t.Fatalf("status=%s error=%v", current.Status, err)
	}
	if err := worker.Work(ctx, testJob(file, 2, 5)); err != nil {
		t.Fatal(err)
	}
	if !store.deleted || calls != 1 || store.deletes != 2 {
		t.Fatalf("deleted=%v scans=%d deletes=%d", store.deleted, calls, store.deletes)
	}
}

func TestCleanScanCallsNextStage(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	pipeline := &ScanProcessor{Storage: &scanStore{data: []byte("image")}, Logger: logger,
		Scanner: scannerFunc(func(context.Context, io.Reader) (scan.Result, error) { return scan.Result{Verdict: scan.Clean}, nil }),
		Next: processorFunc(func(ctx context.Context, q *db.Queries, f db.File) error {
			if f.Status != "processing" {
				t.Errorf("next stage status=%s", f.Status)
			}
			_, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: f.ID, StorageKey: "final/" + f.ID.String(), Mime: f.Mime, Size: f.Size})
			return err
		})}
	worker := &ProcessFile{Pool: pool, Logger: logger, Processor: pipeline}
	if err := worker.Work(ctx, testJob(file, 1, 5)); err != nil {
		t.Fatal(err)
	}
	current, err := db.New(pool).GetFile(ctx, file.ID)
	if err != nil || current.Status != "ready" {
		t.Fatalf("status=%s error=%v", current.Status, err)
	}
}
