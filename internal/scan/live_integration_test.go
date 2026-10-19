package scan

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveClamd(t *testing.T) {
	address := os.Getenv("TEST_CLAMD_ADDRESS")
	if address == "" {
		t.Skip("TEST_CLAMD_ADDRESS is not set")
	}
	client, err := NewClient(address, 10*time.Second, 25<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		data    []byte
		verdict Verdict
	}{
		{"EICAR", []byte(eicar()), Infected}, {"normal PNG", normalPNG(t), Clean},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Scan(context.Background(), bytes.NewReader(tt.data))
			if err != nil || result.Verdict != tt.verdict {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}
