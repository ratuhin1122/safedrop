package httpx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A lost commit acknowledgment may mean PostgreSQL committed successfully.
// Preserve the incoming object in that case rather than break a committed row.
var ErrUploadCommitUncertain = errors.New("upload commit outcome is uncertain")

type UploadRepository interface {
	CreateUpload(context.Context, db.CreateFileParams) (db.File, error)
}

type UploadQueue interface {
	Enqueue(context.Context, pgx.Tx, string) error
}

type UploadDatabase struct {
	Pool  *pgxpool.Pool
	Queue UploadQueue
}

func (d *UploadDatabase) CreateUpload(ctx context.Context, params db.CreateFileParams) (db.File, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return db.File{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	file, err := db.New(tx).CreateFile(ctx, params)
	if err != nil {
		return db.File{}, err
	}
	if err := d.Queue.Enqueue(ctx, tx, file.ID.String()); err != nil {
		return db.File{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			return db.File{}, err
		}
		return db.File{}, fmt.Errorf("%w: %w", ErrUploadCommitUncertain, err)
	}
	return file, nil
}
