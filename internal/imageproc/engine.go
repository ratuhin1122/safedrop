package imageproc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"safedrop/internal/storage"
)

var ErrNativeUnavailable = errors.New("govips requires libvips and a supported CGO build; use Docker on Windows")

type Options struct {
	AVIF      bool
	MaxBytes  int64
	MaxPixels uint64
	TempDir   string
}

type Dimensions struct{ Width, Height int32 }
type Variant struct {
	Dimensions
	TargetWidth int32
	Key, MIME   string
	Size        int64
}

// Encoder writes a complete encoded file to output without whole-file Go buffers.
type Encoder interface {
	Encode(context.Context, string, string, int, bool, uint64) (Dimensions, error)
}

type Engine struct {
	options Options
	encoder Encoder
}

func New(options Options) (*Engine, error) {
	encoder, err := nativeEncoder()
	if err != nil {
		return nil, err
	}
	return NewWithEncoder(options, encoder)
}

func NewWithEncoder(options Options, encoder Encoder) (*Engine, error) {
	if encoder == nil || options.MaxBytes < 1 || options.MaxBytes == 1<<63-1 || options.MaxPixels == 0 {
		return nil, errors.New("encoder, positive bounded upload size, and positive pixel limit are required")
	}
	return &Engine{options, encoder}, nil
}

func (e *Engine) Generate(ctx context.Context, store storage.Store, incoming, prefix string) ([]Variant, error) {
	if !strings.HasPrefix(incoming, "incoming/") || !strings.HasPrefix(prefix, "final/") || !fs.ValidPath(prefix) || strings.ContainsAny(prefix, "\\:\x00") {
		return nil, errors.New("incoming source and final destination are required")
	}
	dir, err := os.MkdirTemp(e.options.TempDir, "safedrop-image-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "source")
	if err := spool(ctx, store, incoming, input, e.options.MaxBytes); err != nil {
		return nil, err
	}
	extension, mime := "webp", "image/webp"
	if e.options.AVIF {
		extension, mime = "avif", "image/avif"
	}
	variants := make([]Variant, 0, 3)
	for _, width := range []int{375, 768, 1200} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		output := filepath.Join(dir, fmt.Sprintf("%d.%s", width, extension))
		dimensions, err := e.encoder.Encode(ctx, input, output, width, e.options.AVIF, e.options.MaxPixels)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s/%d.%s", prefix, width, extension)
		size, err := upload(ctx, store, key, output, mime)
		if err != nil {
			return nil, err
		}
		variants = append(variants, Variant{dimensions, int32(width), key, mime, size})
	}
	return variants, nil
}
