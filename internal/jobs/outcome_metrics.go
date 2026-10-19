package jobs

import (
	"context"
	"safedrop/internal/observability"
)

func recordProcessed(ctx context.Context, status string) {
	result := ""
	switch status {
	case "ready":
		result = "ready"
	case "rejected":
		result = "rejected_error"
	}
	report, _ := ctx.Value(reportKey{}).(*processReport)
	if report != nil {
		if status == "rejected" && report.virus {
			result = "rejected_virus"
		}
		report.result = result
	}
	if result != "" {
		observability.DefaultMetrics.Processed.WithLabelValues(result).Inc()
	}
}
