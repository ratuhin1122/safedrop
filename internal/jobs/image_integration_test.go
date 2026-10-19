package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/imageproc"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

type imageTestEncoder struct{ calls int }

func (e *imageTestEncoder) Encode(_ context.Context, _, output string, width int, _ bool, _ uint64) (imageproc.Dimensions, error) {
	e.calls++
	err := os.WriteFile(output, []byte(fmt.Sprintf("image-%d", width)), 0600)
	return imageproc.Dimensions{Width: int32(width), Height: int32(width / 2)}, err
}

func TestImagePipelineCommitsVariantsAndCleansOriginal(t *testing.T) {
	ctx, pool, file, logger, _ := workerFixture(t)
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Put(ctx, file.StorageKey, strings.NewReader("source"), "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	encoder := &imageTestEncoder{}
	engine, err := imageproc.NewWithEncoder(imageproc.Options{MaxBytes: 1024, MaxPixels: 20000000}, encoder)
	if err != nil {
		t.Fatal(err)
	}
	images := &ImageProcessor{Engine: engine, Storage: store, Pool: pool}
	pipeline := &ScanProcessor{Storage: store, Logger: logger, Next: images,
		Scanner: scannerFunc(func(context.Context, io.Reader) (scan.Result, error) { return scan.Result{Verdict: scan.Clean}, nil })}
	worker := &ProcessFile{Pool: pool, Logger: logger, Processor: pipeline}
	if err := worker.Work(ctx, testJob(file, 1, 5)); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	current, err := q.GetFile(ctx, file.ID)
	if err != nil || current.Status != "ready" || current.Width.Int32 != 1200 || current.Height.Int32 != 600 || current.Size <= 0 {
		t.Fatalf("file=%+v error=%v", current, err)
	}
	variants, err := q.ListFileVariants(ctx, file.ID)
	if err != nil || len(variants) != 3 {
		t.Fatalf("variants=%d error=%v", len(variants), err)
	}
	if body, err := store.Get(ctx, file.StorageKey); !errors.Is(err, storage.ErrNotFound) {
		if body != nil {
			body.Close()
		}
		t.Fatalf("original remains: %v", err)
	}
	if err := worker.Work(ctx, testJob(file, 2, 5)); err != nil {
		t.Fatal(err)
	}
	if encoder.calls != 3 {
		t.Fatalf("re-encoded ready file: %d calls", encoder.calls)
	}
}
