package jobs

import (
	"errors"
	"os"
	"testing"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/scan"

	"github.com/riverqueue/river"
)

func TestWorkerLiveClamd(t *testing.T) {
	address := os.Getenv("TEST_CLAMD_ADDRESS")
	if address == "" {
		t.Skip("TEST_CLAMD_ADDRESS is not set")
	}
	eicar, png := scanFixtures(t)
	for _, tt := range []struct {
		name     string
		data     []byte
		infected bool
	}{
		{"EICAR rejected and deleted", eicar, true}, {"normal image passes", png, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, pool, file, logger, _ := workerFixture(t)
			scanner, err := scan.NewClient(address, 5*time.Second, 25<<20)
			if err != nil {
				t.Fatal(err)
			}
			store := &scanStore{data: tt.data}
			pipeline := &ScanProcessor{Scanner: scanner, Storage: store, Logger: logger}
			worker := &ProcessFile{Pool: pool, Logger: logger, Processor: pipeline}
			err = worker.Work(ctx, testJob(file, 1, 1))
			current, getErr := db.New(pool).GetFile(ctx, file.ID)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if tt.infected {
				if err != nil || current.Status != "rejected" || !store.deleted {
					t.Fatalf("status=%s deleted=%v error=%v", current.Status, store.deleted, err)
				}
				return
			}
			var snoozed *river.JobSnoozeError
			if !errors.As(err, &snoozed) || current.Status != "processing" || store.deleted {
				t.Fatalf("status=%s deleted=%v error=%v", current.Status, store.deleted, err)
			}
			// Processing retries must not rescan or consume the attempt budget.
			if err := worker.Work(ctx, testJob(file, 1, 1)); !errors.As(err, &snoozed) || store.reads != 1 {
				t.Fatalf("clean retry reads=%d error=%v", store.reads, err)
			}
		})
	}
}
