package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestS3MoveDeletesOnlyAfterUpload(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "upload failure"
		}
		t.Run(name, func(t *testing.T) {
			var uploaded, deleted atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					if r.URL.Path != "/private/incoming/file" {
						t.Errorf("source: %s", r.URL.Path)
					}
					w.Header().Set("Content-Type", "text/plain")
					_, _ = io.WriteString(w, "content")
				case http.MethodPut:
					if fail {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					n, err := io.Copy(io.Discard, r.Body)
					if err != nil || n != 7 || r.URL.Path != "/private/final/file" || r.Header.Get("Content-Type") != "text/plain" {
						t.Errorf("upload: %d bytes %v, %s, %s", n, err, r.URL.Path, r.Header.Get("Content-Type"))
					}
					uploaded.Store(true)
				case http.MethodDelete:
					if !uploaded.Load() {
						t.Error("deleted before upload completed")
					}
					deleted.Store(true)
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			err := testS3(t, server.URL).Move(context.Background(), "incoming/file", "final/file")
			if fail && (err == nil || deleted.Load()) {
				t.Fatalf("failed upload lost source: %v %v", err, deleted.Load())
			}
			if !fail && (err != nil || !deleted.Load()) {
				t.Fatalf("move: %v %v", err, deleted.Load())
			}
		})
	}
}

func TestS3MissingObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "<Error><Code>NoSuchKey</Code></Error>")
	}))
	defer server.Close()
	s := testS3(t, server.URL)
	if _, err := s.Get(context.Background(), "incoming/missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing object: %v", err)
	}
	if err := s.Delete(context.Background(), "incoming/missing"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}
