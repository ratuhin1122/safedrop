package config

import (
	"testing"
	"time"
)

func TestWorkerConfig(t *testing.T) {
	for _, tt := range []struct {
		key, value string
		wantErr    bool
	}{
		{"", "", false}, {"DATABASE_URL", "", true}, {"LOG_LEVEL", "invalid", true},
		{"WORKER_CONCURRENCY", "0", true}, {"WORKER_CONCURRENCY", "65", true},
		{"WORKER_CONCURRENCY", "invalid", true}, {"WORKER_CONCURRENCY", "4", false},
		{"WORKER_JOB_TIMEOUT", "0s", true}, {"WORKER_JOB_TIMEOUT", "invalid", true},
		{"SHUTDOWN_TIMEOUT", "-1s", true},
		{"CLAMD_TIMEOUT", "0s", true}, {"MAX_UPLOAD_BYTES", "0", true},
		{"STORAGE_BACKEND", "unknown", true},
	} {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			cfg, err := loadWorker(func(key string) (string, bool) {
				if key == tt.key {
					return tt.value, true
				}
				if key == "DATABASE_URL" {
					return "postgres://localhost/test", true
				}
				return "", false
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: %v", err)
			}
			if tt.key == "" && (cfg.Concurrency != 2 || cfg.JobTimeout != time.Minute || cfg.ShutdownTimeout != 10*time.Second) {
				t.Fatalf("defaults: %+v", cfg)
			}
		})
	}
}
