package jobs

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

const ProcessFileMaxAttempts = 5

type ProcessFileArgs struct {
	FileID      string `json:"file_id"`
	TraceParent string `json:"traceparent,omitempty"`
}

func (ProcessFileArgs) Kind() string { return "process_file" }

func (ProcessFileArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{MaxAttempts: ProcessFileMaxAttempts, Queue: river.QueueDefault}
}

// FileArgs preserves the upload producer's existing argument type and wire format.
type FileArgs = ProcessFileArgs

// Queue is an insert-only River client. Worker execution is configured separately.
type Queue struct{ client *river.Client[pgx.Tx] }

func NewQueue(pool *pgxpool.Pool, logger *slog.Logger) (*Queue, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Logger: logger})
	if err != nil {
		return nil, err
	}
	return &Queue{client: client}, nil
}

func (q *Queue) Enqueue(ctx context.Context, tx pgx.Tx, fileID string) error {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	_, err := q.client.InsertTx(ctx, tx, FileArgs{FileID: fileID, TraceParent: carrier.Get("traceparent")}, nil)
	return err
}
