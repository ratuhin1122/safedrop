package scan

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestINSTREAMVerdicts(t *testing.T) {
	address := mockClamd(t, func(conn net.Conn) {
		size, digest, err := readStream(conn)
		if err != nil {
			t.Error(err)
			return
		}
		reply := "stream: OK\x00"
		if size == 68 && digest == "44d88612fea8a8f36de82e1278abb02f" {
			reply = "stream: Eicar-Test-Signature FOUND\x00"
		}
		_, _ = io.WriteString(conn, reply)
	})
	client, err := NewClient(address, time.Second, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		data    []byte
		verdict Verdict
	}{
		{"EICAR", []byte(eicar()), Infected}, {"normal image", normalPNG(t), Clean},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Scan(context.Background(), bytes.NewReader(tt.data))
			if err != nil || result.Verdict != tt.verdict {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if result.Verdict == Infected && result.Signature == "" {
				t.Fatal("missing signature")
			}
		})
	}
}

func TestMalformedAndErrorRepliesFailClosed(t *testing.T) {
	for _, tt := range []struct {
		name, reply string
		want        error
	}{
		{"daemon error", "stream: scan limit exceeded ERROR\x00", ErrDaemon},
		{"missing terminator", "stream: OK", ErrProtocol},
		{"empty", "", ErrProtocol},
		{"unknown", "something: OK\x00", ErrProtocol},
		{"empty signature", "stream:  FOUND\x00", ErrProtocol},
		{"oversized reply", strings.Repeat("x", 5000) + "\x00", ErrProtocol},
	} {
		t.Run(tt.name, func(t *testing.T) {
			address := mockClamd(t, func(conn net.Conn) { _, _, _ = readStream(conn); _, _ = io.WriteString(conn, tt.reply) })
			client, _ := NewClient(address, time.Second, 1024)
			result, err := client.Scan(context.Background(), strings.NewReader("normal"))
			if !errors.Is(err, tt.want) || result.Verdict != 0 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}

func TestClientConfigurationAndCancellation(t *testing.T) {
	for _, tt := range []struct {
		address string
		timeout time.Duration
		max     int64
	}{
		{"", time.Second, 100}, {"localhost:bad", time.Second, 100}, {"localhost:3310", 0, 100}, {"localhost:3310", time.Second, 0},
	} {
		if _, err := NewClient(tt.address, tt.timeout, tt.max); err == nil {
			t.Fatal("accepted invalid configuration")
		}
	}
	client, _ := NewClient("127.0.0.1:1", time.Second, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := client.Scan(ctx, strings.NewReader("test")); !errors.Is(err, context.Canceled) || result.Verdict != 0 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
