package observability

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSeparateMetricsListenerAndShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := serveMetrics(listener, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer stop()
	client := &http.Client{Timeout: time.Second}
	url := "http://" + listener.Addr().String()
	response, err := client.Get(url + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), "go_goroutines") {
		t.Fatalf("status=%d err=%v", response.StatusCode, err)
	}
	stop()
	if response, err := client.Get(url + "/metrics"); err == nil {
		response.Body.Close()
		t.Fatal("listener still serves after shutdown")
	}
}
