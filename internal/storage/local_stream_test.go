package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type zeroReader struct{ maxRead int }

func (r *zeroReader) Read(p []byte) (int, error) {
	if len(p) > r.maxRead {
		r.maxRead = len(p)
	}
	clear(p)
	return len(p), nil
}

func TestLocalStreamsLargeObject(t *testing.T) {
	l, ctx := newTestLocal(t), context.Background()
	const size = 12 << 20
	source := &zeroReader{}
	if err := l.Put(ctx, "incoming/large", io.LimitReader(source, size), ""); err != nil {
		t.Fatal(err)
	}
	if source.maxRead > 64<<10 {
		t.Fatalf("unbounded read buffer: %d", source.maxRead)
	}
	body, err := l.Get(ctx, "incoming/large")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	n, err := io.Copy(io.Discard, body)
	if err != nil || n != size {
		t.Fatalf("streamed %d bytes: %v", n, err)
	}
}

func TestLocalCanceledOperations(t *testing.T) {
	l := newTestLocal(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Put(ctx, "incoming/file", &zeroReader{}, ""); !errors.Is(err, context.Canceled) {
		t.Errorf("Put: %v", err)
	}
	if _, err := l.Get(ctx, "incoming/file"); !errors.Is(err, context.Canceled) {
		t.Errorf("Get: %v", err)
	}
	if err := l.Move(ctx, "incoming/file", "final/file"); !errors.Is(err, context.Canceled) {
		t.Errorf("Move: %v", err)
	}
	if err := l.Delete(ctx, "incoming/file"); !errors.Is(err, context.Canceled) {
		t.Errorf("Delete: %v", err)
	}
}

func TestLocalSymlinkCannotEscapeRoot(t *testing.T) {
	l := newTestLocal(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(l.root.Name(), "incoming", "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	ctx := context.Background()
	if _, err := l.Get(ctx, "incoming/escape/secret"); err == nil {
		t.Fatal("read outside root")
	}
	if err := l.Put(ctx, "incoming/escape/new", &zeroReader{}, ""); err == nil {
		t.Fatal("wrote outside root")
	}
	if err := l.Delete(ctx, "incoming/escape/secret"); err == nil {
		t.Fatal("deleted outside root")
	}
}
