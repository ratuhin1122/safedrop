package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type unreadBody struct{ reads int }

func (b *unreadBody) Read([]byte) (int, error) {
	b.reads++
	return 0, errors.New("body must not be read")
}
func (*unreadBody) Close() error { return nil }

func TestUploadRejectsBeforeReading(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*http.Request)
		status int
	}{
		{"unauthenticated", func(r *http.Request) { r.Header.Del("Authorization") }, 401},
		{"bad token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer invalid") }, 401},
		{"missing filename", func(r *http.Request) { r.Header.Del("X-Filename") }, 400},
		{"path filename", func(r *http.Request) { r.Header.Set("X-Filename", "../file.png") }, 400},
		{"long filename", func(r *http.Request) { r.Header.Set("X-Filename", strings.Repeat("a", 256)) }, 400},
		{"multiple filenames", func(r *http.Request) { r.Header.Add("X-Filename", "other.png") }, 400},
		{"declared too large", func(r *http.Request) { r.ContentLength = 4097 }, 413},
		{"declared empty", func(r *http.Request) { r.ContentLength = 0 }, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := testUploadHandler(t)
			r := uploadRequest(nil)
			body := &unreadBody{}
			r.Body, r.ContentLength = body, -1
			tt.change(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.status || body.reads != 0 {
				t.Fatalf("status=%d reads=%d", w.Code, body.reads)
			}
		})
	}
}
