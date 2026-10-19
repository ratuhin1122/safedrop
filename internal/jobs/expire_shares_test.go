package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type cleanerFunc func(context.Context) error

func (f cleanerFunc) Clean(ctx context.Context) error { return f(ctx) }

func TestExpireSharesWork(t *testing.T) {
	for _, fail := range []bool{false, true} {
		called := false
		worker := &ExpireShares{Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)), Cleaner: cleanerFunc(func(context.Context) error {
			called = true
			if fail {
				return errors.New("storage offline")
			}
			return nil
		})}
		err := worker.Work(context.Background(), nil)
		if !called || (err != nil) != fail {
			t.Fatalf("called=%v error=%v", called, err)
		}
	}
}
