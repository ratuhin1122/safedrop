package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"safedrop/internal/storage"
)

type WorkerScan struct {
	Address            string
	Timeout            time.Duration
	MaxBytes           int64
	Backend, LocalRoot string
	S3                 storage.S3Config
}

func loadWorkerScan(lookup func(string) (string, bool)) (WorkerScan, error) {
	value := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	c := WorkerScan{Address: value("CLAMD_ADDRESS", "localhost:3310"),
		Backend: value("STORAGE_BACKEND", "local"), LocalRoot: value("STORAGE_LOCAL_ROOT", "./data"),
		S3: storage.S3Config{Endpoint: value("STORAGE_ENDPOINT", ""), Region: value("STORAGE_REGION", "us-east-1"),
			Bucket: value("STORAGE_BUCKET", "safedrop"), AccessKeyID: value("STORAGE_ACCESS_KEY_ID", ""),
			SecretAccessKey: value("STORAGE_SECRET_ACCESS_KEY", ""), SessionToken: value("STORAGE_SESSION_TOKEN", "")}}
	if c.Backend != "local" && c.Backend != "s3" {
		return c, fmt.Errorf("STORAGE_BACKEND must be local or s3")
	}
	var err error
	c.Timeout, err = time.ParseDuration(value("CLAMD_TIMEOUT", "30s"))
	if err != nil || c.Timeout <= 0 {
		return c, fmt.Errorf("CLAMD_TIMEOUT must be a positive duration")
	}
	c.MaxBytes, err = strconv.ParseInt(value("MAX_UPLOAD_BYTES", "26214400"), 10, 64)
	if err != nil || c.MaxBytes <= 0 {
		return c, fmt.Errorf("MAX_UPLOAD_BYTES must be positive")
	}
	c.S3.ForcePathStyle, err = strconv.ParseBool(value("STORAGE_FORCE_PATH_STYLE", "true"))
	if err != nil {
		return c, fmt.Errorf("invalid STORAGE_FORCE_PATH_STYLE")
	}
	return c, nil
}
