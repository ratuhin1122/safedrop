package jobs

import (
	"context"
	"log/slog"
	"safedrop/internal/observability"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// River owns river_job's schema, so these cross-schema maintenance statements
// use pgx directly instead of adding River's internal tables to sqlc's schema.
const exhaustedFiles = `deleted_at IS NULL AND EXISTS (
 SELECT 1 FROM river_job j WHERE j.kind = 'process_file' AND j.state = 'discarded'
 AND j.attempt >= j.max_attempts AND j.args->>'file_id' = files.id::text)`

// ReconcileExhausted repairs final failures caused by timeouts, panics, process
// crashes, or database outages that prevented Work from recording rejection.
func ReconcileExhausted(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err := tx.Exec(ctx, "UPDATE files SET status='scanning' WHERE status='uploaded' AND "+exhaustedFiles); err != nil {
		return 0, err
	}
	result, err := tx.Exec(ctx, "UPDATE files SET status='rejected' WHERE status IN ('scanning','processing') AND "+exhaustedFiles)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	observability.DefaultMetrics.Processed.WithLabelValues("rejected_error").Add(float64(result.RowsAffected()))
	return result.RowsAffected(), nil
}

type Maintainer interface {
	Maintain(context.Context, *pgxpool.Pool) error
}

func RunReconciler(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, maintainers ...Maintainer) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
			count, err := ReconcileExhausted(attempt, pool)
			cancel()
			if err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "exhausted job reconciliation failed", "error", err)
			}
			if err == nil && count > 0 {
				logger.InfoContext(ctx, "exhausted files rejected", "count", count)
			}
			for _, maintainer := range maintainers {
				attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
				err := maintainer.Maintain(attempt, pool)
				cancel()
				if err != nil && ctx.Err() == nil {
					logger.ErrorContext(ctx, "file cleanup maintenance failed", "error", err)
				}
			}
		}
	}
}
