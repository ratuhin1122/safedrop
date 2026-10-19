package jobs

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type FileArgs struct {
	FileID string `json:"file_id"`
}

func (FileArgs) Kind() string { return "process_file" }

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
	_, err := q.client.InsertTx(ctx, tx, FileArgs{FileID: fileID}, nil)
	return err
}
