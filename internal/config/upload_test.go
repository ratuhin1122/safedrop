package config

import "testing"

func TestUploadConfig(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"defaults", nil, false},
		{"limits", map[string]string{"MIN_UPLOAD_BYTES": "10", "MAX_UPLOAD_BYTES": "100", "MAX_IMAGE_PIXELS": "200"}, false},
		{"backend", map[string]string{"STORAGE_BACKEND": "unknown"}, true},
		{"min", map[string]string{"MIN_UPLOAD_BYTES": "0"}, true},
		{"max", map[string]string{"MAX_UPLOAD_BYTES": "0"}, true},
		{"max syntax", map[string]string{"MAX_UPLOAD_BYTES": "many"}, true},
		{"pixels", map[string]string{"MAX_IMAGE_PIXELS": "0"}, true},
		{"path style", map[string]string{"STORAGE_FORCE_PATH_STYLE": "maybe"}, true},
		{"partial stub", map[string]string{"STUB_AUTH_TOKEN": "secret"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadUpload(func(key string) (string, bool) { value, ok := tt.env[key]; return value, ok })
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: %v", err)
			}
			if tt.name == "defaults" && (cfg.Limits.MaxBytes != 26214400 || cfg.Limits.MaxPixels != 20000000 || cfg.Backend != "local") {
				t.Fatalf("defaults: %+v", cfg)
			}
			if tt.name == "limits" && (cfg.Limits.MinBytes != 10 || cfg.Limits.MaxBytes != 100 || cfg.Limits.MaxPixels != 200) {
				t.Fatalf("limits: %+v", cfg.Limits)
			}
		})
	}
}
