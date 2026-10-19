package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/observability"
	"safedrop/internal/ratelimit"
	"safedrop/internal/share"
)

func TestHTTPMetricRouteCardinality(t *testing.T) {
	m := observability.NewMetrics()
	h, _ := testUploadHandler(t)
	router := NewRouter(h.Logger, nil, m, WithRateLimits(ratelimit.NewBuckets(1, 1), m), func(r chi.Router) {
		r.Get("/v1/files/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	})
	for _, path := range []string{"/v1/files/private-id-one?token=secret", "/v1/files/private-id-two", "/unknown-secret"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if strings.HasPrefix(family.GetName(), "http_") {
			for _, metric := range family.Metric {
				for _, label := range metric.Label {
					if label.GetName() == "route" && label.GetValue() != "/v1/files/{id}" && label.GetValue() != "unmatched" {
						t.Fatalf("raw route label: %s", label.GetValue())
					}
				}
			}
		}
	}
	if got := metricValue(t, m, "safedrop_ratelimit_hits_total", "limiter", "api_bucket"); got != 2 {
		t.Fatal(got)
	}
	if got := metricValue(t, m, "http_requests_total", "status", "429"); got != 2 {
		t.Fatalf("unmatched 429 series = %g", got)
	}
}

func TestShareOutcomeMetrics(t *testing.T) {
	m := observability.NewMetrics()
	h := NewShareHandler(ShareHandler{}, m)
	for result, err := range map[string]error{"ok": nil, "expired": share.ErrExpired, "limit_reached": share.ErrLimitReached, "wrong_password": share.ErrPassword, "not_found": share.ErrGone} {
		h.recordShareRequest(err)
		if got := metricValue(t, m, "safedrop_share_requests_total", "result", result); got != 1 {
			t.Fatalf("%s=%g", result, got)
		}
	}
}
