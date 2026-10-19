package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type ShareCleaner interface{ Clean(context.Context) error }
type ExpireSharesArgs struct{}

func (ExpireSharesArgs) Kind() string { return "expire_shares" }

type ExpireShares struct {
	river.WorkerDefaults[ExpireSharesArgs]
	Cleaner ShareCleaner
	Logger  *slog.Logger
}

func (w *ExpireShares) Work(ctx context.Context, _ *river.Job[ExpireSharesArgs]) error {
	if err := w.Cleaner.Clean(ctx); err != nil {
		return err
	}
	w.Logger.InfoContext(ctx, "expired share cleanup completed")
	return nil
}

func (*ExpireShares) NextRetry(job *river.Job[ExpireSharesArgs]) time.Time {
	return time.Now().Add(time.Second << min(max(job.Attempt, 1), 6))
}
