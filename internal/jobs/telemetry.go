package jobs

import (
	"context"
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

func QueueDepth(pool *pgxpool.Pool) observability.QueueDepth {
	return func(ctx context.Context) (int64, error) {
		var count int64
		err := pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE state IN ('available','scheduled','retryable')`).Scan(&count)
		return count, err
	}
}

func (w *ProcessFile) Work(ctx context.Context, job *river.Job[ProcessFileArgs]) (err error) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": job.Args.TraceParent})
	ctx, span := otel.Tracer("safedrop/jobs").Start(ctx, "process_file", trace.WithSpanKind(trace.SpanKindConsumer))
	if job.JobRow != nil {
		span.SetAttributes(attribute.Int64("job.id", job.ID), attribute.Int("job.attempt", job.Attempt))
	}
	started := time.Now()
	defer func() {
		observability.DefaultMetrics.ObserveProcessing(time.Since(started))
		if err != nil {
			span.SetStatus(codes.Error, "processing attempt failed")
		}
		span.End()
	}()
	return w.work(ctx, job)
}
