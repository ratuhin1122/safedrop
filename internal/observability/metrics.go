package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	Registry                               *prometheus.Registry
	Uploads, Processed, RateLimits, Shares *prometheus.CounterVec
	UploadBytes                            prometheus.Counter
	Processing, HTTPDuration               *prometheus.HistogramVec
	HTTPRequests                           *prometheus.CounterVec
	QueueJobs                              *prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	counter := func(name, help string, labels ...string) *prometheus.CounterVec {
		return prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, labels)
	}
	m := &Metrics{Registry: prometheus.NewRegistry(),
		Uploads:      counter("safedrop_uploads_total", "Upload outcomes.", "result"),
		UploadBytes:  prometheus.NewCounter(prometheus.CounterOpts{Name: "safedrop_upload_bytes_total", Help: "Bytes of committed uploads."}),
		Processed:    counter("safedrop_files_processed_total", "Committed processing outcomes and retry attempts.", "result"),
		RateLimits:   counter("safedrop_ratelimit_hits_total", "Denied requests by limiter.", "limiter"),
		Shares:       counter("safedrop_share_requests_total", "Share download authorization outcomes.", "result"),
		Processing:   prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "safedrop_processing_duration_seconds", Help: "Processing attempt duration by stage.", Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120}}, []string{"stage"}),
		HTTPRequests: counter("http_requests_total", "HTTP requests by route template.", "method", "route", "status"),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP duration by route template.", Buckets: prometheus.DefBuckets}, []string{"method", "route"}),
		QueueJobs:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "safedrop_queue_jobs", Help: "River jobs by state across queues."}, []string{"state"}),
	}
	m.Registry.MustRegister(m.Uploads, m.UploadBytes, m.Processed, m.RateLimits, m.Shares, m.Processing, m.HTTPRequests, m.HTTPDuration, m.QueueJobs,
		collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	for _, v := range []string{"accepted", "rejected_size", "rejected_type", "rejected_pixels", "rate_limited", "quota_exceeded"} {
		m.Uploads.WithLabelValues(v)
	}
	for _, v := range []string{"ready", "rejected_virus", "rejected_error", "retried"} {
		m.Processed.WithLabelValues(v)
	}
	for _, v := range []string{"upload_window", "api_bucket", "concurrency"} {
		m.RateLimits.WithLabelValues(v)
	}
	for _, v := range []string{"ok", "expired", "limit_reached", "wrong_password", "not_found"} {
		m.Shares.WithLabelValues(v)
	}
	for _, v := range []string{"scan", "convert", "total"} {
		m.Processing.WithLabelValues(v)
	}
	for _, state := range queueStates {
		m.QueueJobs.WithLabelValues(state)
	}
	return m
}

func (m *Metrics) ObserveProcessing(stage string, elapsed time.Duration) {
	if m == nil {
		return
	}
	m.Processing.WithLabelValues(stage).Observe(elapsed.Seconds())
}

// ObserveUpload charges bytes only for accepted uploads, using the actual size.
func (m *Metrics) ObserveUpload(result string, bytes int64) {
	if m == nil {
		return
	}
	switch result {
	case "accepted", "rejected_size", "rejected_type", "rejected_pixels", "rate_limited", "quota_exceeded":
		m.Uploads.WithLabelValues(result).Inc()
	default:
		return
	}
	if result == "accepted" && bytes > 0 {
		m.UploadBytes.Add(float64(bytes))
	}
}

func (m *Metrics) ObserveProcessed(result string) {
	if m == nil {
		return
	}
	switch result {
	case "ready", "rejected_virus", "rejected_error", "retried":
		m.Processed.WithLabelValues(result).Inc()
	}
}
