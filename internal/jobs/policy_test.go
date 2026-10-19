package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"safedrop/internal/db"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type processorFunc func(context.Context, *db.Queries, db.File) error

func (f processorFunc) Process(ctx context.Context, q *db.Queries, file db.File) error {
	return f(ctx, q, file)
}

func TestRetryPolicy(t *testing.T) {
	w := &ProcessFile{}
	for _, tt := range []struct {
		attempt int
		delay   time.Duration
	}{
		{1, 2 * time.Second}, {2, 4 * time.Second}, {3, 8 * time.Second}, {4, 16 * time.Second}, {30, time.Minute},
	} {
		before := time.Now()
		next := w.NextRetry(&river.Job[ProcessFileArgs]{JobRow: &rivertype.JobRow{Attempt: tt.attempt}})
		after := time.Now()
		if next.Before(before.Add(tt.delay)) || next.After(after.Add(tt.delay)) {
			t.Fatalf("attempt %d: %v", tt.attempt, next)
		}
	}
	if opts := (ProcessFileArgs{}).InsertOpts(); opts.MaxAttempts != 5 || opts.Queue != river.QueueDefault {
		t.Fatalf("insert defaults: %+v", opts)
	}
}

func TestInvalidFileIDCancelsBeforeDatabaseAccess(t *testing.T) {
	for _, id := range []string{"", "invalid", "00000000-0000-0000-0000-000000000000"} {
		err := (&ProcessFile{}).Work(context.Background(), &river.Job[ProcessFileArgs]{Args: ProcessFileArgs{FileID: id}})
		var canceled *river.JobCancelError
		if !errors.As(err, &canceled) {
			t.Fatalf("ID %q: %v", id, err)
		}
	}
}

func TestProcessorFailure(t *testing.T) {
	if err := runProcessor(context.Background(), nil, nil, db.File{}); !errors.Is(err, ErrPipelineUnavailable) {
		t.Fatalf("missing pipeline: %v", err)
	}
	err := runProcessor(context.Background(), processorFunc(func(context.Context, *db.Queries, db.File) error { panic("broken processor") }), nil, db.File{})
	if err == nil {
		t.Fatal("processor panic escaped failure handling")
	}
}
