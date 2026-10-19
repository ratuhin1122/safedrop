package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestTrustedProxy(t *testing.T) {
	for _, tt := range []struct {
		name, peer, forwarded, want string
	}{
		{"untrusted peer", "192.0.2.1:80", "198.51.100.1", "192.0.2.1:80"},
		{"trusted peer", "10.0.0.1:80", "198.51.100.1", "198.51.100.1"},
		{"spoofed left address", "10.0.0.1:80", "192.0.2.99, 198.51.100.1", "198.51.100.1"},
		{"trusted chain", "10.0.0.1:80", "198.51.100.1, 10.0.0.2", "198.51.100.1"},
		{"invalid header", "10.0.0.1:80", "invalid", "10.0.0.1:80"},
		{"missing header", "10.0.0.1:80", "", "10.0.0.1:80"},
		{"ipv6", "[::1]:80", "2001:db8::1", "2001:db8::1"},
		{"mapped ipv4", "[::ffff:10.0.0.1]:80", "198.51.100.1", "198.51.100.1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.peer
			r.Header.Set("X-Forwarded-For", tt.forwarded)
			h := TrustedProxy([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("::1/128")})(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					if r.RemoteAddr != tt.want {
						t.Errorf("client = %q, want %q", r.RemoteAddr, tt.want)
					}
				}))
			h.ServeHTTP(httptest.NewRecorder(), r)
		})
	}
}

func TestNoTrustedProxies(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	TrustedProxy(nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "127.0.0.1:1234" {
			t.Fatal("trusted header by default")
		}
	})).ServeHTTP(httptest.NewRecorder(), r)
}
