package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTurnstile(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"success", `{"success":true,"hostname":"upload.test"}`, 200, nil},
		{"rejected", `{"success":false,"error-codes":["timeout-or-duplicate"]}`, 200, ErrChallenge},
		{"wrong host", `{"success":true,"hostname":"other.test"}`, 200, ErrChallenge},
		{"malformed", `{"success":`, 200, ErrChallengeUnavailable},
		{"service error", `{}`, 503, ErrChallengeUnavailable},
		{"trailing", `{"success":true,"hostname":"upload.test"}{}`, 200, ErrChallengeUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.ParseForm() != nil || r.Form.Get("secret") != "test-secret" ||
					r.Form.Get("response") != "test-token" || r.Form.Get("remoteip") != "192.0.2.1" {
					t.Error("invalid verification request")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			verifier := NewTurnstile("test-secret", "upload.test")
			verifier.endpoint, verifier.client = server.URL, server.Client()
			if err := verifier.Verify(context.Background(), "test-token", "192.0.2.1"); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	verifier := NewTurnstile("test-secret", "")
	for _, token := range []string{"", strings.Repeat("t", 2049)} {
		if err := verifier.Verify(context.Background(), token, "192.0.2.1"); !errors.Is(err, ErrChallenge) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifier.Verify(ctx, "token", "192.0.2.1"); !errors.Is(err, ErrChallengeUnavailable) {
		t.Fatal(err)
	}
}
