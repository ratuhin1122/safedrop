package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var queueStates = []string{"available", "running", "retryable", "discarded"}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{EnableOpenMetrics: false})
}
