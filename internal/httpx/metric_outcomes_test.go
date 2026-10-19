package httpx

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/observability"
	"safedrop/internal/ratelimit"
)

func metricValue(t *testing.T, m *observability.Metrics, name, key, value string) float64 {
	t.Helper()
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name {
			for _, metric := range family.Metric {
				if key == "" {
					return metric.GetCounter().GetValue()
				}
				for _, label := range metric.Label {
					if label.GetName() == key && label.GetValue() == value {
						return metric.GetCounter().GetValue()
					}
				}
			}
		}
	}
	return 0
}

type quotaReader struct{}

func (quotaReader) GetUploadUsage(context.Context, db.GetUploadUsageParams) (db.UploadUsage, error) {
	return db.UploadUsage{Bytes: 4096}, nil
}

func TestUploadOutcomeMetrics(t *testing.T) {
	old := observability.DefaultMetrics
	m := observability.NewMetrics()
	observability.DefaultMetrics = m
	defer func() { observability.DefaultMetrics = old }()
	h, _ := testUploadHandler(t)
	h.Repository = uploadRepositoryFunc(func(_ context.Context, p db.CreateFileParams) (db.File, error) { return db.File{ID: p.OwnerID}, nil })
	data := uploadPNG(t)
	send := func(body []byte, want int) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, uploadRequest(body))
		if w.Code != want {
			t.Fatalf("%d: %s", w.Code, w.Body)
		}
	}
	send(data, 201)
	h.Limits.MaxBytes = 1
	send(data, 413)
	h.Limits.MaxBytes = 4096
	send([]byte("plain text masquerading as an image"), 415)
	h.Limits.MaxPixels = 1
	send(data, 400)
	h.Limits.MaxPixels = 100
	h.Quota = &ratelimit.Quota{Queries: quotaReader{}, Bytes: 4096}
	send(data, 429)
	h.Quota = nil
	u := ratelimit.NewUploads(ratelimit.UploadConfig{Window: time.Minute, PerUser: 1, PerIP: 1})
	h.Admission = u
	send(data, 201)
	send(data, 429)
	u = ratelimit.NewUploads(ratelimit.UploadConfig{Window: time.Minute, PerUser: 20, PerIP: 20})
	h.Admission = u
	for i := 0; i < 3; i++ {
		release, err := u.Begin(context.Background(), uploadTestUser, ratelimit.IP(uploadRequest(data)))
		if err != nil {
			t.Fatal(err)
		}
		defer release()
	}
	send(data, 429)
	for result, want := range map[string]float64{"accepted": 2, "rejected_size": 1, "rejected_type": 1, "rejected_pixels": 1, "quota_exceeded": 1, "rate_limited": 2} {
		if got := metricValue(t, m, "safedrop_uploads_total", "result", result); got != want {
			t.Errorf("%s = %g want %g", result, got, want)
		}
	}
	if got := metricValue(t, m, "safedrop_upload_bytes_total", "", ""); got != float64(2*len(data)) {
		t.Fatal(got)
	}
	for _, kind := range []string{"upload_window", "concurrency"} {
		if got := metricValue(t, m, "safedrop_ratelimit_hits_total", "limiter", kind); got != 1 {
			t.Fatalf("%s = %g", kind, got)
		}
	}
}
