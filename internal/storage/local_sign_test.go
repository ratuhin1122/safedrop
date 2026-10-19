package storage

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLocalSignedDownload(t *testing.T) {
	l, err := NewLocal(LocalConfig{Root: t.TempDir(), BaseURL: "https://example.test/download",
		SigningKey: []byte(strings.Repeat("s", 32))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	ctx := context.Background()
	if err := l.Put(ctx, "final/file", strings.NewReader("content"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	signed, err := l.PresignGet(ctx, "final/file", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		w := httptest.NewRecorder()
		l.ServeHTTP(w, httptest.NewRequest(method, signed, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("download: %d %s", w.Code, w.Body.String())
		}
		if method == http.MethodGet && w.Body.String() != "content" {
			t.Fatalf("body: %q", w.Body.String())
		}
		if w.Header().Get("Content-Disposition") != "attachment" {
			t.Fatal("missing download disposition")
		}
	}
	for _, change := range []func(*url.URL){
		func(u *url.URL) { q := u.Query(); q.Set("key", "final/other"); u.RawQuery = q.Encode() },
		func(u *url.URL) { q := u.Query(); q.Set("expires", "9999999999"); u.RawQuery = q.Encode() },
		func(u *url.URL) {
			q := u.Query()
			expires := strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)
			q.Set("expires", expires)
			q.Set("signature", hex.EncodeToString(l.signature("final/file", expires)))
			u.RawQuery = q.Encode()
		},
	} {
		u, _ := url.Parse(signed)
		change(u)
		w := httptest.NewRecorder()
		l.ServeHTTP(w, httptest.NewRequest(http.MethodGet, u.String(), nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("tampered/expired download: %d", w.Code)
		}
	}
	if _, err := l.PresignGet(ctx, "incoming/file", time.Minute); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("presigned incoming: %v", err)
	}
}

func TestLocalSigningConfiguration(t *testing.T) {
	l := newTestLocal(t)
	if _, err := l.PresignGet(context.Background(), "final/file", time.Minute); !errors.Is(err, ErrPresignDisabled) {
		t.Fatalf("unsigned URL: %v", err)
	}
	for _, ttl := range []time.Duration{0, -time.Second, 8 * 24 * time.Hour} {
		if _, err := l.PresignGet(context.Background(), "final/file", ttl); err == nil {
			t.Fatal("invalid TTL accepted")
		}
	}
	for _, cfg := range []LocalConfig{
		{},
		{Root: t.TempDir(), BaseURL: "file:///tmp/files"},
		{Root: t.TempDir(), BaseURL: "https://example.test", SigningKey: []byte("short")},
	} {
		if store, err := NewLocal(cfg); err == nil {
			store.Close()
			t.Fatal("invalid configuration accepted")
		}
	}
}
