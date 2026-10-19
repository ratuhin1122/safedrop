package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Worker struct {
	DatabaseURL                 string
	LogLevel                    slog.Level
	Concurrency                 int
	JobTimeout, ShutdownTimeout time.Duration
	Scan                        WorkerScan
}

func LoadWorker() (Worker, error) { return loadWorker(os.LookupEnv) }

func loadWorker(lookup func(string) (string, bool)) (Worker, error) {
	value := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	c := Worker{DatabaseURL: value("DATABASE_URL", "")}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required for the worker")
	}
	if err := c.LogLevel.UnmarshalText([]byte(value("LOG_LEVEL", "INFO"))); err != nil {
		return c, fmt.Errorf("invalid LOG_LEVEL")
	}
	var err error
	c.Concurrency, err = strconv.Atoi(value("WORKER_CONCURRENCY", "2"))
	if err != nil || c.Concurrency < 1 || c.Concurrency > 64 {
		return c, fmt.Errorf("WORKER_CONCURRENCY must be between 1 and 64")
	}
	c.JobTimeout, err = time.ParseDuration(value("WORKER_JOB_TIMEOUT", "1m"))
	if err != nil || c.JobTimeout <= 0 {
		return c, fmt.Errorf("WORKER_JOB_TIMEOUT must be a positive duration")
	}
	c.ShutdownTimeout, err = time.ParseDuration(value("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || c.ShutdownTimeout <= 0 {
		return c, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration")
	}
	c.Scan, err = loadWorkerScan(lookup)
	return c, err
}
