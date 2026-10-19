package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestJSONLogger(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(&output, slog.LevelInfo)
	logger.Debug("hidden")
	if output.Len() != 0 {
		t.Fatal("debug message escaped level filter")
	}
	logger.Info("started", "service", "api")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["level"] != "INFO" || entry["msg"] != "started" || entry["service"] != "api" {
		t.Fatalf("unexpected entry: %v", entry)
	}
}
