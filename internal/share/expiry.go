package share

import (
	"context"
	"errors"

	"safedrop/internal/db"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Cleaner struct {
	Pool    *pgxpool.Pool
	Storage storage.Store
}

func (c *Cleaner) Clean(ctx context.Context) error {
	q := db.New(c.Pool)
	files, err := q.ListExpiredShareFiles(ctx)
	if err != nil {
		return err
	}
	var failure error
	for _, id := range files {
		if err := c.expire(ctx, id); err != nil && failure == nil {
			failure = err
		}
	}
	pending, err := q.ListShareFileDeletions(ctx)
	if err != nil {
		return errors.Join(failure, err)
	}
	for _, id := range pending {
		if err := q.TouchShareFileDeletion(ctx, id); err != nil {
			return errors.Join(failure, err)
		}
		if err := c.deleteObjects(ctx, id); err != nil && failure == nil {
			failure = err
		}
	}
	return failure
}

func (c *Cleaner) expire(ctx context.Context, id pgtype.UUID) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	q := db.New(tx)
	if _, err := q.LockExpiryFile(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if err := q.DeleteExpiredFileLinks(ctx, id); err != nil {
		return err
	}
	active, err := q.HasActiveShareLinks(ctx, id)
	if err != nil {
		return err
	}
	if !active {
		if err := q.MarkFileExpired(ctx, id); err != nil {
			return err
		}
		if err := q.ScheduleShareFileDeletion(ctx, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
