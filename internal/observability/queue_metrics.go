package observability

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type QueueStates func(context.Context) (map[string]int64, error)
type queueCollector struct {
	query QueueStates
	desc  *prometheus.Desc
}

var queueStates = []string{"available", "running", "retryable", "discarded"}

func newQueueCollector(query QueueStates) *queueCollector {
	return &queueCollector{query: query, desc: prometheus.NewDesc("safedrop_queue_jobs", "River jobs by state across queues.", []string{"state"}, nil)}
}
func (c *queueCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c *queueCollector) Collect(ch chan<- prometheus.Metric) {
	counts := map[string]int64{}
	if c.query != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var err error
		counts, err = c.query(ctx)
		cancel()
		if err != nil {
			ch <- prometheus.NewInvalidMetric(c.desc, err)
			return
		}
	}
	for _, state := range queueStates {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(counts[state]), state)
	}
}
