package observability

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type queryFunc func(context.Context, string, ...any) (pgx.Rows, error)

func (f queryFunc) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return f(ctx, sql, args...)
}

type countRows struct {
	pgx.Rows
	states       []string
	counts       []int64
	index        int
	err, scanErr error
	closed       chan struct{}
}

func (r *countRows) Next() bool { return r.index < len(r.states) }
func (r *countRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	*dest[0].(*string) = r.states[r.index]
	*dest[1].(*int64) = r.counts[r.index]
	r.index++
	return nil
}
func (r *countRows) Err() error { return r.err }
func (r *countRows) Close() {
	if r.closed != nil {
		close(r.closed)
	}
}

func TestQueuePollResetsMissingStatesAndContinuesAfterError(t *testing.T) {
	m := NewMetrics()
	var logs bytes.Buffer
	first, last := make(chan struct{}), make(chan struct{})
	calls := 0
	pool := queryFunc(func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		calls++
		if sql != queueCountsSQL || len(args) != 0 {
			t.Error("expected one grouped query")
		}
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
			t.Error("missing query timeout")
		}
		switch calls {
		case 1:
			return &countRows{states: []string{"available", "running"}, counts: []int64{5, 2}, closed: first}, nil
		case 2:
			return nil, errors.New("forced query failure")
		default:
			return &countRows{closed: last}, nil
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticks := make(chan time.Time, 2)
	done := make(chan struct{})
	go func() { defer close(done); runQueue(ctx, pool, m, slog.New(slog.NewTextHandler(&logs, nil)), ticks) }()
	select {
	case <-first:
	case <-time.After(2 * time.Second):
		t.Fatal("initial poll did not run")
	}
	if got := testutil.ToFloat64(m.QueueJobs.WithLabelValues("available")); got != 5 {
		t.Fatal(got)
	}
	ticks <- time.Now()
	ticks <- time.Now()
	select {
	case <-last:
	case <-time.After(2 * time.Second):
		t.Fatal("poller did not recover")
	}
	cancel()
	<-done
	if calls != 3 || !strings.Contains(logs.String(), "forced query failure") {
		t.Fatalf("calls=%d logs=%s", calls, &logs)
	}
	for _, state := range queueStates {
		if got := testutil.ToFloat64(m.QueueJobs.WithLabelValues(state)); got != 0 {
			t.Fatalf("%s=%g", state, got)
		}
	}
}

func TestQueuePartialResultsDoNotReplaceSnapshot(t *testing.T) {
	for _, scanFailure := range []bool{false, true} {
		m := NewMetrics()
		m.QueueJobs.WithLabelValues("available").Set(7)
		rows := &countRows{states: []string{"available"}, counts: []int64{1}}
		if scanFailure {
			rows.scanErr = errors.New("scan failed")
		} else {
			rows.err = errors.New("iteration failed")
		}
		err := refreshQueue(context.Background(), queryFunc(func(context.Context, string, ...any) (pgx.Rows, error) { return rows, nil }), m)
		if err == nil || testutil.ToFloat64(m.QueueJobs.WithLabelValues("available")) != 7 {
			t.Fatal("partial result replaced snapshot")
		}
	}
}

func TestQueueCancellationStopsActiveQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	pool := queryFunc(func(ctx context.Context, _ string, _ ...any) (pgx.Rows, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	stop := StartQueue(ctx, pool, NewMetrics(), slog.Default())
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		cancel()
		stop()
		t.Fatal("query did not start")
	}
	cancel()
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("poller did not stop")
	}
}
