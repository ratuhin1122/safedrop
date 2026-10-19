package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuckets(t *testing.T) {
	b := NewBuckets(2, 2)
	now := time.Unix(100, 0)
	b.now = func() time.Time { return now }
	if b.Allow("a") != 0 || b.Allow("a") != 0 || b.Allow("a") < 500*time.Millisecond {
		t.Fatal("burst not enforced")
	}
	if b.Allow("b") != 0 {
		t.Fatal("separate IP blocked")
	}
	now = now.Add(500 * time.Millisecond)
	if b.Allow("a") != 0 {
		t.Fatal("refill failed")
	}
}

func TestIPAndRetryAfter(t *testing.T) {
	for _, address := range []string{"[::ffff:192.0.2.1]:123", "192.0.2.1:123", "192.0.2.1"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = address
		r.Header.Set("X-Forwarded-For", "198.51.100.1")
		if IP(r) != "192.0.2.1" {
			t.Fatal(IP(r))
		}
	}
	w := httptest.NewRecorder()
	Respond(w, 1500*time.Millisecond)
	if w.Code != 429 || w.Header().Get("Retry-After") != "2" {
		t.Fatal(w.Result())
	}
}
