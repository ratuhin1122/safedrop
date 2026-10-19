package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"
)

// Config contains API settings read from the process environment.
type Config struct {
	HTTPAddr        string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	TrustedProxies  []netip.Prefix
	Upload          Upload
}

func Load() (Config, error) { return load(os.LookupEnv) }

func load(lookup func(string) (string, bool)) (Config, error) {
	value := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	c := Config{HTTPAddr: value("HTTP_ADDR", ":8080")}
	if c.HTTPAddr == "" {
		return c, fmt.Errorf("HTTP_ADDR must not be empty")
	}
	if err := c.LogLevel.UnmarshalText([]byte(value("LOG_LEVEL", "INFO"))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	var err error
	c.ShutdownTimeout, err = time.ParseDuration(value("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || c.ShutdownTimeout <= 0 {
		return c, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration")
	}
	if raw := value("TRUSTED_PROXY_CIDRS", ""); raw != "" {
		for _, cidr := range strings.Split(raw, ",") {
			prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
			if err != nil {
				return c, fmt.Errorf("TRUSTED_PROXY_CIDRS: %w", err)
			}
			c.TrustedProxies = append(c.TrustedProxies, prefix.Masked())
		}
	}
	c.Upload, err = loadUpload(lookup)
	return c, err
}
