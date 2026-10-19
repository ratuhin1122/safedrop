package observability

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	m := &Metrics{}
	m.Uploads.Add(2)
	m.UploadRejections.Add(1)
	m.Rejected.Add(3)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.ObserveProcessing(time.Second) }()
	}
	wg.Wait()
	w := httptest.NewRecorder()
	m.Handler(func(context.Context) (int64, error) { return 7, nil }).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain; version=0.0.4") {
		t.Fatal(w.Result())
	}
	for _, want := range []string{"safedrop_uploads_total 2\n", "safedrop_rejected_total 3\n", "safedrop_upload_rejections_total 1\n",
		"safedrop_queue_depth 7\n", "safedrop_queue_depth_up 1\n", "safedrop_processing_duration_seconds_bucket{le=\"0.5\"} 0\n",
		"safedrop_processing_duration_seconds_bucket{le=\"1\"} 10\n", "safedrop_processing_duration_seconds_bucket{le=\"+Inf\"} 10\n",
		"safedrop_processing_duration_seconds_count 10\n", "safedrop_processing_duration_seconds_sum 10\n"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	w = httptest.NewRecorder()
	m.Handler(func(context.Context) (int64, error) { return 0, errors.New("offline") }).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), "safedrop_queue_depth_up 0\n") {
		t.Fatal(w.Body)
	}
}
