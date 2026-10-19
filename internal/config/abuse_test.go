package config

import "testing"

func TestAbuseConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"UPLOAD_RATE_WINDOW", "0s"}, {"UPLOAD_RATE_PER_USER", "0"}, {"UPLOAD_RATE_PER_IP", "bad"},
		{"DAILY_UPLOAD_BYTES", "-1"}, {"RATE_LIMIT_REQUESTS_PER_SECOND", "NaN"},
		{"RATE_LIMIT_REQUESTS_PER_SECOND", "+Inf"}, {"RATE_LIMIT_BURST", "0"}, {"TURNSTILE_HOSTNAME", "example.test"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			_, err := loadAbuse(func(key string) (string, bool) { return tc.value, key == tc.key })
			if err == nil {
				t.Fatal("accepted invalid setting")
			}
		})
	}
	a, err := loadAbuse(func(string) (string, bool) { return "", false })
	if err != nil || a.Uploads.PerUser != 20 || a.Uploads.PerIP != 20 || a.DailyBytes != 1<<30 {
		t.Fatalf("defaults: %+v %v", a, err)
	}
}
