package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
)

type usageFunc func(context.Context, db.GetUploadUsageParams) (db.UploadUsage, error)

func (f usageFunc) GetUploadUsage(c context.Context, p db.GetUploadUsageParams) (db.UploadUsage, error) {
	return f(c, p)
}

func TestQuota(t *testing.T) {
	for _, tc := range []struct {
		name       string
		used, size int64
		err        error
		denied     bool
	}{
		{"boundary", 90, 10, nil, false}, {"over", 90, 11, nil, true},
		{"unknown length exhausted", 100, -1, nil, true}, {"new day", 0, 10, pgx.ErrNoRows, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := Quota{Bytes: 100, Queries: usageFunc(func(_ context.Context, p db.GetUploadUsageParams) (db.UploadUsage, error) {
				if p.Day.Time.Location() != time.UTC {
					t.Fatal("day is not UTC")
				}
				return db.UploadUsage{Bytes: tc.used}, tc.err
			})}
			err := q.Check(context.Background(), pgtype.UUID{}, tc.size)
			var denied *Rejection
			if errors.As(err, &denied) != tc.denied {
				t.Fatal(err)
			}
		})
	}
	if got := UntilTomorrow(time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC)); got != time.Minute {
		t.Fatal(got)
	}
}
