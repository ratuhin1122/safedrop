package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	router := NewRouter(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK || w.Body.String() != "{\"status\":\"ok\"}\n" || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("health response: %d %s %s", w.Code, w.Header(), w.Body.String())
	}
}
