package jobs

import (
	"errors"
	"log/slog"
	"safedrop/internal/observability"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type WorkerOptions struct {
	Concurrency int
	JobTimeout  time.Duration
	// Empty selects River's default queue. A separate queue can isolate consumers.
	Queue        string
	ShareCleaner ShareCleaner
	Metrics      *observability.Metrics
}

func NewWorkerClient(pool *pgxpool.Pool, logger *slog.Logger, options WorkerOptions, processor Processor) (*river.Client[pgx.Tx], error) {
	if pool == nil || logger == nil || options.Concurrency < 1 || options.Concurrency > 64 || options.JobTimeout <= 0 {
		return nil, errors.New("pool, logger, 1-64 workers, and a positive job timeout are required")
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, NewProcessFile(pool, processor, logger, options.JobTimeout, options.Metrics))
	queue := options.Queue
	if queue == "" {
		queue = river.QueueDefault
	}
	var periodic []*river.PeriodicJob
	if options.ShareCleaner != nil {
		river.AddWorker(workers, &ExpireShares{Cleaner: options.ShareCleaner, Logger: logger})
		periodic = []*river.PeriodicJob{river.NewPeriodicJob(river.PeriodicInterval(time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return ExpireSharesArgs{}, &river.InsertOpts{Queue: queue, MaxAttempts: 5}
			},
			&river.PeriodicJobOpts{RunOnStart: true})}
	}
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: logger, Workers: workers, MaxAttempts: ProcessFileMaxAttempts,
		PeriodicJobs: periodic,
		Queues:       map[string]river.QueueConfig{queue: {MaxWorkers: options.Concurrency}},
	})
}
