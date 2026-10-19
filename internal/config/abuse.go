package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"safedrop/internal/ratelimit"
)

type Abuse struct {
	Uploads                            ratelimit.UploadConfig
	DailyBytes                         int64
	PerSecond                          float64
	Burst                              int
	TurnstileSecret, TurnstileHostname string
}

func loadAbuse(lookup func(string) (string, bool)) (Abuse, error) {
	value := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	a := Abuse{TurnstileSecret: value("TURNSTILE_SECRET", ""), TurnstileHostname: value("TURNSTILE_HOSTNAME", "")}
	var err error
	a.Uploads.Window, err = time.ParseDuration(value("UPLOAD_RATE_WINDOW", "1m"))
	if err != nil || a.Uploads.Window < time.Second || a.Uploads.Window > 24*time.Hour {
		return a, fmt.Errorf("invalid UPLOAD_RATE_WINDOW")
	}
	for _, field := range []struct {
		key, fallback string
		target        *int
	}{
		{"UPLOAD_RATE_PER_USER", "20", &a.Uploads.PerUser}, {"UPLOAD_RATE_PER_IP", "20", &a.Uploads.PerIP},
		{"RATE_LIMIT_BURST", "10", &a.Burst},
	} {
		*field.target, err = strconv.Atoi(value(field.key, field.fallback))
		if err != nil || *field.target < 1 || *field.target > 10000 {
			return a, fmt.Errorf("invalid %s", field.key)
		}
	}
	a.DailyBytes, err = strconv.ParseInt(value("DAILY_UPLOAD_BYTES", "1073741824"), 10, 64)
	if err != nil || a.DailyBytes < 1 {
		return a, fmt.Errorf("DAILY_UPLOAD_BYTES must be positive")
	}
	a.PerSecond, err = strconv.ParseFloat(value("RATE_LIMIT_REQUESTS_PER_SECOND", "5"), 64)
	if err != nil || math.IsNaN(a.PerSecond) || math.IsInf(a.PerSecond, 0) || a.PerSecond < 0.001 || a.PerSecond > 10000 {
		return a, fmt.Errorf("invalid RATE_LIMIT_REQUESTS_PER_SECOND")
	}
	if a.TurnstileHostname != "" && a.TurnstileSecret == "" {
		return a, fmt.Errorf("TURNSTILE_HOSTNAME requires TURNSTILE_SECRET")
	}
	return a, nil
}
