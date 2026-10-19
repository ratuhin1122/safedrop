package observability

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
)

func TestRequestLoggingOneJSONLine(t *testing.T) {
	for _, incoming := range []string{"", "client-request_123", "invalid/request", strings.Repeat("x", 65)} {
		t.Run(incoming, func(t *testing.T) {
			var output bytes.Buffer
			router := chi.NewRouter()
			router.Use(RequestLogging(NewLogger(&output, slog.LevelInfo)))
			router.Post("/s/{token}", func(w http.ResponseWriter, r *http.Request) {
				if RequestID(r.Context()) == "" {
					t.Error("missing context ID")
				}
				SetUserID(r.Context(), "user-123")
				w.WriteHeader(201)
			})
			r := httptest.NewRequest("POST", "/s/private-share-token?password=private-password", strings.NewReader("private-file-contents"))
			r.Header.Set("X-Request-ID", incoming)
			r.Header.Set("Authorization", "Bearer private-auth-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			id := w.Header().Get("X-Request-ID")
			if !validRequestID(id) || (validRequestID(incoming) && id != incoming) {
				t.Fatalf("request ID %q", id)
			}
			raw := output.String()
			decoder := json.NewDecoder(&output)
			var entry map[string]any
			if err := decoder.Decode(&entry); err != nil {
				t.Fatal(err)
			}
			if err := decoder.Decode(&struct{}{}); err != io.EOF {
				t.Fatalf("more than one log line: %v", err)
			}
			for key, want := range map[string]any{"request_id": id, "method": "POST", "route": "/s/{token}", "status": float64(201), "user_id": "user-123"} {
				if entry[key] != want {
					t.Errorf("%s=%v", key, entry[key])
				}
			}
			if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
				t.Fatal("invalid duration")
			}
			for _, secret := range []string{"private-share-token", "private-password", "private-file-contents", "private-auth-token", "Authorization"} {
				if strings.Contains(raw, secret) {
					t.Fatalf("sensitive value logged: %s", secret)
				}
			}
		})
	}
}

func TestLoggerLevelFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "WARN")
	var output bytes.Buffer
	logger, err := NewLoggerFromEnv(&output)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hidden")
	logger.Warn("visible")
	if strings.Contains(output.String(), "hidden") || !strings.Contains(output.String(), `"level":"WARN"`) {
		t.Fatal(output.String())
	}
	t.Setenv("LOG_LEVEL", "invalid")
	if _, err := NewLoggerFromEnv(&output); err == nil {
		t.Fatal("invalid level accepted")
	}
}

func TestUnmatchedPathIsNotLogged(t *testing.T) {
	var output bytes.Buffer
	router := chi.NewRouter()
	router.Use(RequestLogging(NewLogger(&output, slog.LevelInfo)))
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/unknown-secret", nil))
	if strings.Contains(output.String(), "unknown-secret") || !strings.Contains(output.String(), `"route":"unmatched"`) {
		t.Fatal(output.String())
	}
}
