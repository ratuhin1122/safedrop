package observability

import (
	"github.com/prometheus/client_golang/prometheus/testutil"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetricsRegistryAndScrape(t *testing.T) {
	m := NewMetrics()
	m.ObserveUpload("accepted", 20)
	m.ObserveUpload("accepted", 22)
	m.ObserveProcessed("rejected_virus")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.ObserveProcessing("scan", time.Second) }()
	}
	wg.Wait()
	w := httptest.NewRecorder()
	m.QueueJobs.WithLabelValues("available").Set(7)
	m.QueueJobs.WithLabelValues("running").Set(2)
	m.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
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
	other.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), `safedrop_uploads_total{result="accepted"} 0`) {
		t.Fatal("registries share counters")
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
		"safedrop_queue_jobs":                  {"state": "available running retryable discarded"},
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

func TestObservationHelpers(t *testing.T) {
	m := NewMetrics()
	for _, result := range []string{"accepted", "rejected_size", "rejected_type", "rejected_pixels", "rate_limited", "quota_exceeded"} {
		m.ObserveUpload(result, 128)
		if got := testutil.ToFloat64(m.Uploads.WithLabelValues(result)); got != 1 {
			t.Errorf("upload %s=%g", result, got)
		}
	}
	for _, result := range []string{"ready", "rejected_virus", "rejected_error", "retried"} {
		m.ObserveProcessed(result)
		if got := testutil.ToFloat64(m.Processed.WithLabelValues(result)); got != 1 {
			t.Errorf("processed %s=%g", result, got)
		}
	}
	m.ObserveUpload("private-user-id", 1000)
	m.ObserveProcessed("private-file-id")
	m.ObserveUpload("accepted", -1)
	if got := testutil.ToFloat64(m.UploadBytes); got != 128 {
		t.Fatalf("accepted bytes=%g", got)
	}
	if got := testutil.ToFloat64(m.Uploads.WithLabelValues("accepted")); got != 2 {
		t.Fatal(got)
	}
	if count := testutil.CollectAndCount(m.Uploads); count != 6 {
		t.Fatalf("upload labels: %d", count)
	}
	if count := testutil.CollectAndCount(m.Processed); count != 4 {
		t.Fatalf("processed labels: %d", count)
	}
	other := NewMetrics()
	if got := testutil.ToFloat64(other.UploadBytes); got != 0 {
		t.Fatal("registries share state")
	}
}
