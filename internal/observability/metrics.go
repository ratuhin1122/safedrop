package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics emits Prometheus text format without unbounded request labels.
type Metrics struct {
	Uploads          atomic.Uint64
	UploadRejections atomic.Uint64
	Rejected         atomic.Uint64
	mu               sync.Mutex
	count            uint64
	sum              float64
	buckets          [8]uint64
}

var DefaultMetrics = &Metrics{}
var boundaries = [...]float64{0.1, 0.5, 1, 5, 10, 30, 60, 120}

type QueueDepth func(context.Context) (int64, error)

func (m *Metrics) ObserveProcessing(elapsed time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	seconds := elapsed.Seconds()
	m.count++
	m.sum += seconds
	for i, bound := range boundaries {
		if seconds <= bound {
			m.buckets[i]++
		}
	}
}

func (m *Metrics) Handler(depth QueueDepth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queue, up := int64(0), 0
		if depth != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			var err error
			queue, err = depth(ctx)
			cancel()
			if err == nil {
				up = 1
			}
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		for _, item := range []struct {
			name, help string
			value      uint64
		}{
			{"safedrop_uploads_total", "Committed HTTP uploads.", m.Uploads.Load()},
			{"safedrop_upload_rejections_total", "Upload HTTP responses with status 4xx.", m.UploadRejections.Load()},
			{"safedrop_rejected_total", "Files transitioned to rejected by this worker process.", m.Rejected.Load()},
		} {
			fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", item.name, item.help, item.name, item.name, item.value)
		}
		fmt.Fprintf(w, "# HELP safedrop_queue_depth Pending River jobs across queues.\n# TYPE safedrop_queue_depth gauge\nsafedrop_queue_depth %d\n", queue)
		fmt.Fprintf(w, "# HELP safedrop_queue_depth_up Whether the queue query succeeded.\n# TYPE safedrop_queue_depth_up gauge\nsafedrop_queue_depth_up %d\n", up)
		m.mu.Lock()
		count, sum, buckets := m.count, m.sum, m.buckets
		m.mu.Unlock()
		fmt.Fprint(w, "# HELP safedrop_processing_duration_seconds ProcessFile attempt duration including retries.\n# TYPE safedrop_processing_duration_seconds histogram\n")
		for i, bound := range boundaries {
			fmt.Fprintf(w, "safedrop_processing_duration_seconds_bucket{le=\"%g\"} %d\n", bound, buckets[i])
		}
		fmt.Fprintf(w, "safedrop_processing_duration_seconds_bucket{le=\"+Inf\"} %d\nsafedrop_processing_duration_seconds_sum %g\nsafedrop_processing_duration_seconds_count %d\n", count, sum, count)
	})
}
