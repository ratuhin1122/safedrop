package jobs

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"io"
	"testing"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/observability"
	"safedrop/internal/scan"
)

func processedCount(t *testing.T, m *observability.Metrics, result string) float64 {
	t.Helper()
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() == "safedrop_files_processed_total" {
			for _, metric := range f.Metric {
				for _, label := range metric.Label {
					if label.GetValue() == result {
						return metric.GetCounter().GetValue()
					}
				}
			}
		}
	}
	return 0
}

func TestProcessingOutcomeMetrics(t *testing.T) {
	for _, result := range []string{"ready", "rejected_virus", "rejected_error", "retried"} {
		t.Run(result, func(t *testing.T) {
			ctx, pool, file, logger, _ := workerFixture(t)
			m := observability.NewMetrics()
			var processor Processor = processorFunc(readyProcessor)
			attempt := 1
			if result == "rejected_error" || result == "retried" {
				processor = processorFunc(func(context.Context, *db.Queries, db.File) error { return errors.New("conversion failed") })
				if result == "rejected_error" {
					attempt = 2
				}
			}
			if result == "rejected_virus" {
				processor = &ScanProcessor{Storage: &scanStore{data: []byte("test"), failDeletes: 1}, Logger: logger,
					Scanner: scannerFunc(func(context.Context, io.Reader) (scan.Result, error) { return scan.Result{Verdict: scan.Infected}, nil })}
			}
			worker := NewProcessFile(pool, processor, logger, 0, m)
			_ = worker.Work(ctx, testJob(file, attempt, 2))
			if got := processedCount(t, m, result); got != 1 {
				t.Fatalf("%s = %g", result, got)
			}
			if result != "retried" {
				_ = worker.Work(ctx, testJob(file, attempt, 2))
				if got := processedCount(t, m, result); got != 1 {
					t.Fatalf("replay counted twice: %g", got)
				}
			}
		})
	}
}

func TestQueueStateMetrics(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	queue, err := NewQueue(pool, logger)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"available", "running", "retryable", "discarded"} {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := queue.Enqueue(ctx, tx, file.ID.String()); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `UPDATE river_job SET state=$1::river_job_state, attempted_at=now(),
 finalized_at=CASE WHEN $1::river_job_state='discarded' THEN now() ELSE NULL END
 WHERE id=(SELECT max(id) FROM river_job WHERE args->>'file_id'=$2)`, state, file.ID.String())
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	m := observability.NewMetrics()
	stop := observability.StartQueue(ctx, pool, m, logger)
	defer stop()
	deadline := time.Now().Add(2 * time.Second)
	for _, state := range []string{"available", "running", "retryable", "discarded"} {
		for testutil.ToFloat64(m.QueueJobs.WithLabelValues(state)) < 1 {
			if time.Now().After(deadline) {
				t.Fatalf("%s gauge was not updated", state)
			}
			time.Sleep(time.Millisecond)
		}
	}
}
