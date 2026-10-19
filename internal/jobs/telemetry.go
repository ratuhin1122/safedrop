package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"safedrop/internal/observability"
)

func QueueMetrics(pool *pgxpool.Pool) observability.QueueStates {
	return func(ctx context.Context) (map[string]int64, error) {
		rows, err := pool.Query(ctx, `SELECT state::text,count(*) FROM river_job WHERE state IN ('available','running','retryable','discarded') GROUP BY state`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		counts := map[string]int64{}
		for rows.Next() {
			var state string
			var count int64
			if err := rows.Scan(&state, &count); err != nil {
				return nil, err
			}
			counts[state] = count
		}
		return counts, rows.Err()
	}
}

type processReport struct {
	virus  bool
	result string
}
type reportKey struct{}

func (w *ProcessFile) Work(ctx context.Context, job *river.Job[ProcessFileArgs]) (err error) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": job.Args.TraceParent})
	ctx, span := otel.Tracer("safedrop/jobs").Start(ctx, "process_file", trace.WithSpanKind(trace.SpanKindConsumer))
	if job.JobRow != nil {
		span.SetAttributes(attribute.Int64("job.id", job.ID), attribute.Int("job.attempt", job.Attempt))
	}
	started := time.Now()
	report := &processReport{}
	ctx = context.WithValue(ctx, reportKey{}, report)
	defer func() {
		observability.DefaultMetrics.ObserveProcessing("total", time.Since(started))
		var canceled *river.JobCancelError
		var snoozed *river.JobSnoozeError
		if err != nil && !errors.As(err, &canceled) && !errors.As(err, &snoozed) && report.result == "" && job.JobRow != nil && job.Attempt < job.MaxAttempts {
			observability.DefaultMetrics.Processed.WithLabelValues("retried").Inc()
		}
		if err != nil {
			span.SetStatus(codes.Error, "processing attempt failed")
		}
		span.End()
	}()
	return w.work(ctx, job)
}
