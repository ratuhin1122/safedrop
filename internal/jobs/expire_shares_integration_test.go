package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/riverqueue/river"
)

func TestPeriodicShareCleanupRunsOnStart(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	called := make(chan struct{}, 1)
	cleaner := cleanerFunc(func(context.Context) error {
		select {
		case called <- struct{}{}:
		default:
		}
		return nil
	})
	client, err := NewWorkerClient(pool, logger, WorkerOptions{Concurrency: 1, JobTimeout: time.Minute,
		Queue: "expiry_test_" + file.ID.String(), ShareCleaner: cleaner}, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := client.Subscribe(river.EventKindJobCompleted)
	defer unsubscribe()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.StopAndCancel(stop)
	}()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("periodic cleanup did not run")
		case event := <-events:
			if event.Job.Kind != "expire_shares" {
				continue
			}
			select {
			case <-called:
				return
			default:
				t.Fatal("periodic worker did not call cleaner")
			}
		}
	}
}
