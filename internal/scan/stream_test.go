package scan

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type zeros struct{ maxRead int }

func (z *zeros) Read(p []byte) (int, error) {
	if len(p) > z.maxRead {
		z.maxRead = len(p)
	}
	clear(p)
	return len(p), nil
}

func TestLargeStreamUsesBoundedChunks(t *testing.T) {
	var received atomic.Int64
	address := mockClamd(t, func(conn net.Conn) {
		n, _, err := readStream(conn)
		if err != nil {
			t.Error(err)
			return
		}
		received.Store(n)
		_, _ = io.WriteString(conn, "stream: OK\x00")
	})
	client, _ := NewClient(address, time.Second, 5<<20)
	source := &zeros{}
	result, err := client.Scan(context.Background(), io.LimitReader(source, 4<<20))
	if err != nil || result.Verdict != Clean || received.Load() != 4<<20 || source.maxRead > 32<<10 {
		t.Fatalf("result=%+v received=%d buffer=%d error=%v", result, received.Load(), source.maxRead, err)
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestStreamFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source io.Reader
		want   error
	}{
		{"over max", strings.NewReader("too large"), ErrTooLarge},
		{"reader failure", brokenReader{}, io.ErrUnexpectedEOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			address := mockClamd(t, func(conn net.Conn) { _, _, _ = readStream(conn) })
			client, _ := NewClient(address, time.Second, 4)
			result, err := client.Scan(context.Background(), tt.source)
			if !errors.Is(err, tt.want) || result.Verdict != 0 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}

func TestUnresponsiveDaemonTimesOut(t *testing.T) {
	address := mockClamd(t, func(conn net.Conn) { _, _, _ = readStream(conn); _, _ = io.Copy(io.Discard, conn) })
	client, _ := NewClient(address, 50*time.Millisecond, 1024)
	started := time.Now()
	result, err := client.Scan(context.Background(), strings.NewReader("test"))
	if err == nil || result.Verdict != 0 || time.Since(started) > time.Second {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
