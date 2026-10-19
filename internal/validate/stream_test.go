package validate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type streamBody struct {
	reader  io.Reader
	maxRead int
	closed  bool
}

func (b *streamBody) Read(p []byte) (int, error) {
	if len(p) > b.maxRead {
		b.maxRead = len(p)
	}
	return b.reader.Read(p)
}
func (b *streamBody) Close() error { b.closed = true; return nil }

type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestStreamingAndClose(t *testing.T) {
	const extra = 12 << 20
	data := pngFixture(t)
	body := &streamBody{reader: io.MultiReader(bytes.NewReader(data), io.LimitReader(zeros{}, extra))}
	r := httptest.NewRequest(http.MethodPost, "/upload", nil)
	r.Body = body
	file, err := Validate(httptest.NewRecorder(), r, Limits{1, int64(len(data)) + extra, 100, t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if body.maxRead > 64<<10 || !body.closed {
		t.Fatalf("read buffer=%d closed=%v", body.maxRead, body.closed)
	}
	if file.Size != int64(len(data))+extra {
		t.Fatalf("size: %d", file.Size)
	}
	// Trailing bytes count toward upload limits even after valid image metadata.
	r = httptest.NewRequest(http.MethodPost, "/upload", io.MultiReader(bytes.NewReader(data), io.LimitReader(zeros{}, extra)))
	if f, err := Validate(httptest.NewRecorder(), r, Limits{1, 1024, 100, t.TempDir()}); !errors.Is(err, ErrTooLarge) {
		if f != nil {
			f.Close()
		}
		t.Fatalf("trailing bytes bypassed limit: %v", err)
	}
}

func TestInvalidLimits(t *testing.T) {
	for _, limits := range []Limits{{}, {MinBytes: -1, MaxBytes: 10, MaxPixels: 10},
		{MinBytes: 10, MaxBytes: 1, MaxPixels: 10}, {MinBytes: 1, MaxBytes: 10}} {
		r := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(pngFixture(t)))
		if file, err := Validate(httptest.NewRecorder(), r, limits); err == nil {
			file.Close()
			t.Fatalf("accepted limits: %+v", limits)
		}
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestRequestFailures(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		dir := t.TempDir()
		r := httptest.NewRequest(http.MethodPost, "/upload", nil)
		body := &streamBody{reader: brokenReader{}}
		r.Body = body
		want := error(io.ErrUnexpectedEOF)
		if canceled {
			ctx, cancel := context.WithCancel(r.Context())
			cancel()
			r = r.WithContext(ctx)
			want = context.Canceled
		}
		if file, err := Validate(httptest.NewRecorder(), r, Limits{1, 1024, 100, dir}); !errors.Is(err, want) {
			if file != nil {
				file.Close()
			}
			t.Fatalf("error: %v, want %v", err, want)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 || !body.closed {
			t.Fatalf("cleanup: %v %v %v", entries, err, body.closed)
		}
	}
}
