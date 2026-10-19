package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
)

type UsageReader interface {
	GetUploadUsage(context.Context, db.GetUploadUsageParams) (db.UploadUsage, error)
}
type Quota struct {
	Queries UsageReader
	Bytes   int64
}

// Check is an early rejection only; the transaction atomically charges actual bytes.
func (q *Quota) Check(ctx context.Context, user pgtype.UUID, size int64) error {
	if q.Bytes <= 0 {
		return nil
	}
	now := time.Now().UTC()
	usage, err := q.Queries.GetUploadUsage(ctx, db.GetUploadUsageParams{UserID: user, Day: pgtype.Date{Time: now, Valid: true}})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if size < 1 {
		size = 1
	}
	if size > q.Bytes || usage.Bytes > q.Bytes-size {
		return &Rejection{RetryAfter: UntilTomorrow(now)}
	}
	return nil
}
