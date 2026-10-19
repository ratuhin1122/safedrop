package share

import (
	"context"
	"errors"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *Cleaner) deleteObjects(ctx context.Context, id pgtype.UUID) error {
	q := db.New(c.Pool)
	file, err := q.GetExpiredFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	keys := map[string]bool{file.StorageKey: true}
	variants, err := q.ListFileVariants(ctx, id)
	if err != nil {
		return err
	}
	for _, variant := range variants {
		keys[variant.StorageKey] = true
	}
	cleanup, err := q.GetFileCleanup(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		keys[cleanup.StorageKey] = true
	}
	for key := range keys {
		if err := c.Storage.Delete(ctx, key); err != nil {
			return err
		}
	}
	return q.DeleteExpiredFile(ctx, id)
}
