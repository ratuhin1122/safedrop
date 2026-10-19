package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddlewareUsesFinalRoute(t *testing.T) {
	m := NewMetrics()
	router := chi.NewRouter()
	router.Use(m.HTTPMiddleware)
	router.Route("/v1", func(r chi.Router) {
		r.Get("/files/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })
	})
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	for _, path := range []string{"/v1/files/private-one", "/v1/files/private-two", "/healthz", "/private-unknown"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	for _, tc := range []struct {
		route, status string
		count         float64
	}{
		{"/v1/files/{id}", "201", 2}, {"/healthz", "200", 1}, {"unmatched", "404", 1},
	} {
		if got := testutil.ToFloat64(m.HTTPRequests.WithLabelValues("GET", tc.route, tc.status)); got != tc.count {
			t.Fatalf("%s: %g", tc.route, got)
		}
	}
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == "http_request_duration_seconds" {
			for _, metric := range family.Metric {
				if metric.GetHistogram().GetSampleCount() == 0 {
					t.Fatal("duration was not observed")
				}
				for _, label := range metric.Label {
					if label.GetName() == "route" && label.GetValue() != "/v1/files/{id}" && label.GetValue() != "/healthz" && label.GetValue() != "unmatched" {
						t.Fatal("raw path label")
					}
				}
			}
		}
	}
}
