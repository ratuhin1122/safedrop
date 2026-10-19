package jobs

import (
	"context"
	"time"
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
		report.metrics.ObserveProcessed(result)
	}
}

func observeStage(ctx context.Context, stage string, elapsed time.Duration) {
	if report, _ := ctx.Value(reportKey{}).(*processReport); report != nil {
		report.metrics.ObserveProcessing(stage, elapsed)
	}
}
