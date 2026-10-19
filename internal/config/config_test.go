package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"defaults", nil, false},
		{"overrides", map[string]string{"HTTP_ADDR": "127.0.0.1:9000", "LOG_LEVEL": "debug",
			"SHUTDOWN_TIMEOUT": "2s", "TRUSTED_PROXY_CIDRS": "10.0.0.0/8, ::1/128"}, false},
		{"bad proxy", map[string]string{"TRUSTED_PROXY_CIDRS": "10.0.0.1"}, true},
		{"bad level", map[string]string{"LOG_LEVEL": "invalid"}, true},
		{"bad duration", map[string]string{"SHUTDOWN_TIMEOUT": "soon"}, true},
		{"zero duration", map[string]string{"SHUTDOWN_TIMEOUT": "0s"}, true},
		{"empty address", map[string]string{"HTTP_ADDR": " "}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := load(func(key string) (string, bool) { v, ok := tt.env[key]; return v, ok })
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load error = %v", err)
			}
			if tt.name == "defaults" && (cfg.HTTPAddr != ":8080" || cfg.ShutdownTimeout != 10*time.Second || len(cfg.TrustedProxies) != 0) {
				t.Fatalf("unexpected defaults: %+v", cfg)
			}
			if tt.name == "overrides" && (cfg.HTTPAddr != "127.0.0.1:9000" || cfg.ShutdownTimeout != 2*time.Second || len(cfg.TrustedProxies) != 2 || cfg.LogLevel.String() != "DEBUG") {
				t.Fatalf("unexpected overrides: %+v", cfg)
			}
		})
	}
}
