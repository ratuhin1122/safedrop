package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type QueueStates func(context.Context) (map[string]int64, error)

var queueStates = []string{"available", "running", "retryable", "discarded"}

func (m *Metrics) Handler(query QueueStates) http.Handler {
	handler := promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{EnableOpenMetrics: false})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if query != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			counts, err := query(ctx)
			cancel()
			if err != nil {
				http.Error(w, "queue metrics unavailable", http.StatusInternalServerError)
				return
			}
			for _, state := range queueStates {
				m.QueueJobs.WithLabelValues(state).Set(float64(counts[state]))
			}
		}
		handler.ServeHTTP(w, r)
	})
}
