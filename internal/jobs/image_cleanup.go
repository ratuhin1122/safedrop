package jobs

import (
	"context"
	"errors"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (p *ImageProcessor) Finalize(ctx context.Context, file db.File) error {
	q := db.New(p.Pool)
	cleanup, err := q.GetFileCleanup(ctx, file.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return p.cleanup(ctx, q, cleanup)
}

func (p *ImageProcessor) cleanup(ctx context.Context, q *db.Queries, cleanup db.FileCleanup) error {
	if err := p.Storage.Delete(ctx, cleanup.StorageKey); err != nil {
		return err
	}
	return q.DeleteFileCleanup(ctx, db.DeleteFileCleanupParams{FileID: cleanup.FileID, StorageKey: cleanup.StorageKey})
}

// Maintain retries committed cleanup records even if River exhausted its retries.
// Keyset pagination bounds memory and prevents a failing object starving others.
func (p *ImageProcessor) Maintain(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	after := pgtype.UUID{Valid: true}
	var failures error
	for {
		batch, err := q.ListFileCleanup(ctx, after)
		if err != nil {
			return errors.Join(failures, err)
		}
		for _, item := range batch {
			if err := p.cleanup(ctx, q, item); err != nil && failures == nil {
				failures = err
			}
			after = item.FileID
		}
		if len(batch) < 100 {
			return failures
		}
	}
}
