package observability_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"safedrop/internal/db"
	"safedrop/internal/jobs"
	"safedrop/internal/observability"
)

func TestFiveEnqueuedJobsWithoutWorker(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := jobs.Migrate(ctx, pool, logger); err != nil {
		t.Fatal(err)
	}
	var baseline int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE state='available'").Scan(&baseline); err != nil {
		t.Fatal(err)
	}
	queue, err := jobs.NewQueue(pool, logger)
	if err != nil {
		t.Fatal(err)
	}
	marker := pgtype.UUID{Valid: true}
	_, _ = rand.Read(marker.Bytes[:])
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanup, "DELETE FROM river_job WHERE args->>'file_id'=$1", marker.String())
	}()
	// NewQueue is insert-only: no worker is started in this test.
	for i := 0; i < 5; i++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := queue.Enqueue(ctx, tx, marker.String()); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	m := observability.NewMetrics()
	stop := observability.StartQueue(ctx, pool, m, logger)
	defer stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for testutil.ToFloat64(m.QueueJobs.WithLabelValues("available")) != float64(baseline+5) {
		select {
		case <-ctx.Done():
			t.Fatal("queue gauge was not refreshed")
		case <-ticker.C:
		}
	}
	server := httptest.NewServer(m.Handler())
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`safedrop_queue_jobs{state="available"} %d`, baseline+5)
	if response.StatusCode != 200 || !strings.Contains(string(body), want) {
		t.Fatalf("missing %s", want)
	}
	t.Log(want)
}
