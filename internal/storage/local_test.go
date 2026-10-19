package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func newTestLocal(t *testing.T) *Local {
	t.Helper()
	l, err := NewLocal(LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	return l
}

func TestLocalLifecycle(t *testing.T) {
	l, ctx := newTestLocal(t), context.Background()
	if err := l.Put(ctx, "incoming/nested/file", strings.NewReader("content"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := l.Move(ctx, "incoming/nested/file", "final/nested/file"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Get(ctx, "incoming/nested/file"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("source: %v", err)
	}
	body, err := l.Get(ctx, "final/nested/file")
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(body)
	closeErr := body.Close()
	if readErr != nil || closeErr != nil || string(got) != "content" {
		t.Fatalf("Get: %q %v %v", got, readErr, closeErr)
	}
	if err := l.Move(ctx, "final/nested/file", "final/nested/file"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := l.Delete(ctx, "final/nested/file"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := l.Get(ctx, "final/nested/file"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted: %v", err)
	}
	if err := l.Move(ctx, "incoming/missing", "final/missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing move: %v", err)
	}
}

func TestLocalInvalidKeys(t *testing.T) {
	l, ctx := newTestLocal(t), context.Background()
	for _, key := range []string{"", "incoming", "final/", "public/file", "/incoming/file", "incoming/../secret",
		"incoming//file", "incoming/./file", `incoming\file`, "incoming/a:b", "final/../../secret", "incoming/a\x00b"} {
		t.Run(key, func(t *testing.T) {
			if err := l.Put(ctx, key, strings.NewReader("x"), ""); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Put: %v", err)
			}
			if _, err := l.Get(ctx, key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Get: %v", err)
			}
			if err := l.Delete(ctx, key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Delete: %v", err)
			}
			if err := l.Move(ctx, key, "final/ok"); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Move source: %v", err)
			}
			if err := l.Move(ctx, "incoming/ok", key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Move target: %v", err)
			}
		})
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("source failure") }

func TestLocalFailedPutKeepsOriginal(t *testing.T) {
	l, ctx := newTestLocal(t), context.Background()
	if err := l.Put(ctx, "incoming/file", strings.NewReader("original"), ""); err != nil {
		t.Fatal(err)
	}
	if err := l.Put(ctx, "incoming/file", io.MultiReader(strings.NewReader("partial"), failedReader{}), ""); err == nil {
		t.Fatal("expected reader failure")
	}
	body, err := l.Get(ctx, "incoming/file")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil || string(got) != "original" {
		t.Fatalf("original changed: %q %v", got, err)
	}
	dir, err := l.root.Open("incoming")
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v %v", entries, err)
	}
}
