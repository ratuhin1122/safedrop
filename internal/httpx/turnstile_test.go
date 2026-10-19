package httpx

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"safedrop/internal/auth"
)

type challengeFunc func(context.Context, string, string) error

func (f challengeFunc) Verify(c context.Context, token, ip string) error { return f(c, token, ip) }

func TestRegistrationChallengeRunsBeforeAccountCreation(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{{auth.ErrChallenge, 403}, {auth.ErrChallengeUnavailable, 503}} {
		called := false
		check := challengeFunc(func(_ context.Context, token, ip string) error {
			called = true
			if token != "challenge" || ip != "192.0.2.1" {
				t.Errorf("token or IP not passed")
			}
			return tc.err
		})
		r := httptest.NewRequest("POST", "/v1/auth/register", strings.NewReader(`{"email":"a@example.test","password":"long password","turnstile_token":"challenge"}`))
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		accountEndpoint(nil, true, check)(w, r)
		if !called || w.Code != tc.status {
			t.Fatalf("called=%v status=%d", called, w.Code)
		}
	}
}
