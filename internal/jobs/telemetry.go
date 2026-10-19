package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"safedrop/internal/observability"
)

type processReport struct {
	virus   bool
	result  string
	metrics *observability.Metrics
}
type reportKey struct{}

func (w *ProcessFile) Work(ctx context.Context, job *river.Job[ProcessFileArgs]) (err error) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": job.Args.TraceParent})
	ctx, span := otel.Tracer("safedrop/jobs").Start(ctx, "process_file", trace.WithSpanKind(trace.SpanKindConsumer))
	if job.JobRow != nil {
		span.SetAttributes(attribute.Int64("job.id", job.ID), attribute.Int("job.attempt", job.Attempt))
	}
	started := time.Now()
	report := &processReport{metrics: w.Metrics}
	ctx = context.WithValue(ctx, reportKey{}, report)
	defer func() {
		w.Metrics.ObserveProcessing("total", time.Since(started))
		var canceled *river.JobCancelError
		var snoozed *river.JobSnoozeError
		if err != nil && !errors.As(err, &canceled) && !errors.As(err, &snoozed) && report.result == "" && job.JobRow != nil && job.Attempt < job.MaxAttempts {
			w.Metrics.ObserveProcessed("retried")
		}
		if err != nil {
			span.SetStatus(codes.Error, "processing attempt failed")
		}
		span.End()
	}()
	return w.work(ctx, job)
}
