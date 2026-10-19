package share

import (
	"context"
	"time"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ Pool *pgxpool.Pool }

func (p *Postgres) Create(ctx context.Context, params db.CreateShareLinkParams) (db.ShareLink, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return db.ShareLink{}, err
	}
	defer rollback(ctx, tx)
	q := db.New(tx)
	file, err := q.LockFile(ctx, params.FileID)
	if err != nil {
		return db.ShareLink{}, err
	}
	if file.OwnerID != params.OwnerID || file.Status != "ready" {
		return db.ShareLink{}, pgx.ErrNoRows
	}
	row, err := q.CreateShareLink(ctx, params)
	if err != nil {
		return db.ShareLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.ShareLink{}, err
	}
	return row, nil
}

func (p *Postgres) Lookup(ctx context.Context, hash []byte) (db.ShareLink, error) {
	return db.New(p.Pool).GetShareLinkState(ctx, hash)
}
func (p *Postgres) File(ctx context.Context, id pgtype.UUID) (db.File, error) {
	return db.New(p.Pool).GetFile(ctx, id)
}
func (p *Postgres) Consume(ctx context.Context, hash []byte) (db.ShareLink, error) {
	return db.New(p.Pool).ConsumeShareDownload(ctx, hash)
}

func rollback(ctx context.Context, tx pgx.Tx) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = tx.Rollback(cleanup)
}
