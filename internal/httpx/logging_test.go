package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/observability"
)

func TestRouterLogsExactlyOneRequestRecord(t *testing.T) {
	for _, tc := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/panic", 500}} {
		var output bytes.Buffer
		router := NewRouter(observability.NewLogger(&output, slog.LevelInfo), nil, nil, func(r chi.Router) {
			r.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("private-file-content") })
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		raw := output.String()
		decoder := json.NewDecoder(&output)
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			t.Fatal("duplicate request log")
		}
		if entry["request_id"] != w.Header().Get("X-Request-ID") || entry["status"] != float64(tc.status) || entry["route"] != tc.path {
			t.Fatal(entry)
		}
		if strings.Contains(raw, "private-file-content") {
			t.Fatal("panic payload leaked")
		}
	}
}

func TestUploadBadTypeWarnsWithoutPayload(t *testing.T) {
	h, _ := testUploadHandler(t)
	var output bytes.Buffer
	h.Logger = observability.NewLogger(&output, slog.LevelInfo)
	router := NewRouter(h.Logger, nil, nil, WithUploads(h))
	r := uploadRequest([]byte("private-file-content"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal(w.Code)
	}
	raw := output.String()
	decoder := json.NewDecoder(&output)
	warnings, requests := 0, 0
	for {
		var entry map[string]any
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if entry["msg"] == "file rejected" {
			warnings++
			if entry["level"] != "WARN" || entry["file_id"] == "" || entry["reason"] != "rejected_type" {
				t.Fatal(entry)
			}
		}
		if entry["msg"] == "http request" {
			requests++
			if entry["user_id"] != uploadTestUser {
				t.Fatal(entry)
			}
		}
	}
	if warnings != 1 || requests != 1 || strings.Contains(raw, "private-file-content") || strings.Contains(raw, uploadTestToken) {
		t.Fatal(raw)
	}
}
