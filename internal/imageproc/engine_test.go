package imageproc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"safedrop/internal/storage"
)

type testEncoder struct {
	fail  bool
	calls int
}

func (e *testEncoder) Encode(_ context.Context, _, output string, width int, _ bool, _ uint64) (Dimensions, error) {
	e.calls++
	if e.fail {
		return Dimensions{}, errors.New("encoding failed")
	}
	err := os.WriteFile(output, []byte(fmt.Sprintf("encoded-%d", width)), 0600)
	return Dimensions{int32(width), int32(width / 2)}, err
}

func TestGenerateStreamsVariants(t *testing.T) {
	for _, avif := range []bool{false, true} {
		store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		ctx := context.Background()
		if err := store.Put(ctx, "incoming/source", strings.NewReader("source"), "image/jpeg"); err != nil {
			t.Fatal(err)
		}
		encoder := &testEncoder{}
		engine, err := NewWithEncoder(Options{AVIF: avif, MaxBytes: 1024, MaxPixels: 20000000, TempDir: t.TempDir()}, encoder)
		if err != nil {
			t.Fatal(err)
		}
		variants, err := engine.Generate(ctx, store, "incoming/source", "final/file")
		if err != nil {
			t.Fatal(err)
		}
		if len(variants) != 3 || encoder.calls != 3 {
			t.Fatalf("variants=%d calls=%d", len(variants), encoder.calls)
		}
		for i, width := range []int32{375, 768, 1200} {
			variant := variants[i]
			if variant.Width != width || variant.TargetWidth != width || variant.Size <= 0 {
				t.Fatalf("variant=%+v", variant)
			}
			body, err := store.Get(ctx, variant.Key)
			if err != nil {
				t.Fatal(err)
			}
			body.Close()
		}
		body, err := store.Get(ctx, "incoming/source")
		if err != nil {
			t.Fatal("original removed before database commit")
		}
		body.Close()
	}
}

func TestGenerateFailurePreservesSource(t *testing.T) {
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.Put(ctx, "incoming/source", strings.NewReader("source"), ""); err != nil {
		t.Fatal(err)
	}
	encoder := &testEncoder{fail: true}
	engine, _ := NewWithEncoder(Options{MaxBytes: 1024, MaxPixels: 100}, encoder)
	if _, err := engine.Generate(ctx, store, "incoming/source", "final/file"); err == nil {
		t.Fatal("expected encoder error")
	}
	body, err := store.Get(ctx, "incoming/source")
	if err != nil {
		t.Fatal(err)
	}
	body.Close()
	engine, _ = NewWithEncoder(Options{MaxBytes: 2, MaxPixels: 100}, encoder)
	if _, err := engine.Generate(ctx, store, "incoming/source", "final/file"); err == nil {
		t.Fatal("accepted oversized source")
	}
}
