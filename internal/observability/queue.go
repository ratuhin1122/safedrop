package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

const queueCountsSQL = `SELECT state::text, count(*) FROM river_job
WHERE state IN ('available','running','retryable','discarded') GROUP BY state`

// QueueDB is satisfied by the existing pgx pool; no extra connection pool is opened.
type QueueDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// StartQueue refreshes immediately and every 15 seconds. Stop waits for the
// goroutine to exit, so the owner can safely close the shared pool afterward.
func StartQueue(parent context.Context, pool QueueDB, metrics *Metrics, logger *slog.Logger) func() {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		runQueue(ctx, pool, metrics, logger, ticker.C)
	}()
	return func() { cancel(); <-done }
}

func runQueue(ctx context.Context, pool QueueDB, metrics *Metrics, logger *slog.Logger, ticks <-chan time.Time) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := refreshQueue(ctx, pool, metrics); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "queue metrics update failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticks:
		}
	}
}

func refreshQueue(parent context.Context, pool QueueDB, metrics *Metrics) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	rows, err := pool.Query(ctx, queueCountsSQL)
	if err != nil {
		return err
	}
	defer rows.Close()
	counts := make(map[string]int64, 4)
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return err
		}
		counts[state] = count
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, state := range queueStates {
		metrics.QueueJobs.WithLabelValues(state).Set(float64(counts[state]))
	}
	return nil
}
