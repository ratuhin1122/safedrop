package db

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUploadUsageAccumulates(t *testing.T) {
	ctx, q := testQueries(t)
	u, err := q.CreateUser(ctx, CreateUserParams{Email: "usage@example.test", PasswordHash: "hash"})
	if err != nil {
		t.Fatal(err)
	}
	day := pgtype.Date{Time: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), Valid: true}
	for i := 0; i < 2; i++ {
		if _, err := q.AddUploadUsage(ctx, AddUploadUsageParams{UserID: u.ID, Day: day, Bytes: 42}); err != nil {
			t.Fatal(err)
		}
	}
	usage, err := q.GetUploadUsage(ctx, GetUploadUsageParams{UserID: u.ID, Day: day})
	if err != nil || usage.Bytes != 84 || usage.Count != 2 {
		t.Fatalf("usage = %+v, error = %v", usage, err)
	}
}
