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

func TestMetricsRegistryAndScrape(t *testing.T) {
	m := NewMetrics()
	m.Uploads.WithLabelValues("accepted").Add(2)
	m.UploadBytes.Add(42)
	m.Processed.WithLabelValues("rejected_virus").Inc()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.ObserveProcessing("scan", time.Second) }()
	}
	wg.Wait()
	w := httptest.NewRecorder()
	m.Handler(func(context.Context) (map[string]int64, error) {
		return map[string]int64{"available": 7, "running": 2}, nil
	}).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
		t.Fatal(w.Result())
	}
	for _, want := range []string{`safedrop_uploads_total{result="accepted"} 2`, "safedrop_upload_bytes_total 42",
		`safedrop_files_processed_total{result="rejected_virus"} 1`, `safedrop_queue_jobs{state="available"} 7`,
		`safedrop_queue_jobs{state="running"} 2`, `safedrop_queue_jobs{state="discarded"} 0`,
		`safedrop_processing_duration_seconds_bucket{stage="scan",le="1"} 10`,
		`safedrop_processing_duration_seconds_count{stage="scan"} 10`, "go_goroutines "} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	other := NewMetrics()
	w = httptest.NewRecorder()
	other.Handler(nil).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), `safedrop_uploads_total{result="accepted"} 0`) {
		t.Fatal("registries share counters")
	}
	w = httptest.NewRecorder()
	m.Handler(func(context.Context) (map[string]int64, error) { return nil, errors.New("offline") }).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 500 {
		t.Fatalf("failed queue scrape returned %d", w.Code)
	}
}

func TestMetricLabelConventions(t *testing.T) {
	m := NewMetrics()
	m.HTTPRequests.WithLabelValues("GET", "/v1/files/{id}", "200").Inc()
	m.HTTPDuration.WithLabelValues("GET", "/v1/files/{id}").Observe(0.1)
	allowed := map[string]map[string]string{
		"safedrop_uploads_total":               {"result": "accepted rejected_size rejected_type rejected_pixels rate_limited quota_exceeded"},
		"safedrop_files_processed_total":       {"result": "ready rejected_virus rejected_error retried"},
		"safedrop_processing_duration_seconds": {"stage": "scan convert total"},
		"safedrop_ratelimit_hits_total":        {"limiter": "upload_window api_bucket concurrency"},
		"safedrop_share_requests_total":        {"result": "ok expired limit_reached wrong_password not_found"},
		"safedrop_upload_bytes_total":          {},
		"http_requests_total":                  {"method": "GET", "route": "/v1/files/{id}", "status": "200"},
		"http_request_duration_seconds":        {"method": "GET", "route": "/v1/files/{id}"},
	}
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		labels, ok := allowed[family.GetName()]
		if !ok {
			continue
		}
		delete(allowed, family.GetName())
		for _, metric := range family.Metric {
			if len(metric.Label) != len(labels) {
				t.Fatalf("unexpected labels for %s", family.GetName())
			}
			for _, label := range metric.Label {
				if !strings.Contains(" "+labels[label.GetName()]+" ", " "+label.GetValue()+" ") {
					t.Fatalf("invalid label %s=%s", label.GetName(), label.GetValue())
				}
			}
		}
	}
	if len(allowed) != 0 {
		t.Fatalf("missing metric families: %v", allowed)
	}
}
