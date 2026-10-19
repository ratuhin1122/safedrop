package jobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"safedrop/internal/db"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestWorkerPicksUpRetriesAndRejects(t *testing.T) {
	ctx, pool, file, logger, logs := workerFixture(t)
	queue := "worker_test_" + file.ID.String()
	client, err := NewWorkerClient(pool, logger, WorkerOptions{Concurrency: 1, JobTimeout: 5 * time.Second, Queue: queue}, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := client.Subscribe(river.EventKindJobFailed)
	defer unsubscribe()
	inserted, err := client.Insert(ctx, ProcessFileArgs{FileID: file.ID.String()}, &river.InsertOpts{MaxAttempts: 2, Queue: queue})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.StopAndCancel(stop); err != nil {
			t.Error(err)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("worker did not exhaust the job")
		case event := <-events:
			if event.Job.ID != inserted.Job.ID {
				continue
			}
			if event.Job.State == rivertype.JobStateDiscarded {
				current, err := db.New(pool).GetFile(ctx, file.ID)
				if err != nil || current.Status != "rejected" || event.Job.Attempt != 2 {
					t.Fatalf("file=%+v job=%+v error=%v", current, event.Job, err)
				}
				if !strings.Contains(logs.String(), `"msg":"process_file started"`) || !strings.Contains(logs.String(), file.ID.String()) {
					t.Fatalf("missing pickup log: %s", logs.String())
				}
				t.Log("worker picked up and logged the job, retried, and rejected after attempt 2")
				return
			}
			current, err := db.New(pool).GetFile(ctx, file.ID)
			if err != nil || current.Status != "scanning" {
				t.Fatalf("retry status=%s error=%v", current.Status, err)
			}
		}
	}
}
