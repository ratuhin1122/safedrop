//go:build cgo && !windows

package imageproc

import (
	"context"
	"encoding/binary"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"safedrop/internal/storage"

	"github.com/davidbyttow/govips/v2/vips"
)

func sampleJPEG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jpg")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(file, image.NewRGBA(image.Rect(0, 0, 1600, 800)), nil); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	output := filepath.Join(dir, "oriented.jpg")
	target, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	var soi [2]byte
	if _, err := io.ReadFull(source, soi[:]); err != nil {
		t.Fatal(err)
	}
	_, _ = target.Write(soi[:])
	exif := []byte("Exif\x00\x00II\x2a\x00\x08\x00\x00\x00\x01\x00\x12\x01\x03\x00\x01\x00\x00\x00\x06\x00\x00\x00\x00\x00\x00\x00")
	segment := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(exif)+2))
	_, _ = target.Write(segment)
	_, _ = target.Write(exif)
	if _, err := io.Copy(target, source); err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	return output
}

func TestNativeJPEGProducesThreeVariants(t *testing.T) {
	for _, avif := range []bool{false, true} {
		name := "WebP"
		if avif {
			name = "AVIF"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			store, err := storage.NewLocal(storage.LocalConfig{Root: root})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			input, err := os.Open(sampleJPEG(t))
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			ctx := context.Background()
			if err := store.Put(ctx, "incoming/sample", input, "image/jpeg"); err != nil {
				t.Fatal(err)
			}
			engine, err := New(Options{AVIF: avif, MaxBytes: 25 << 20, MaxPixels: 20000000})
			if err != nil {
				t.Fatal(err)
			}
			variants, err := engine.Generate(ctx, store, "incoming/sample", "final/sample")
			if err != nil {
				t.Fatal(err)
			}
			if len(variants) != 3 {
				t.Fatalf("variants=%d", len(variants))
			}
			for _, variant := range variants {
				if variant.Width != variant.TargetWidth || variant.Height != 2*variant.Width {
					t.Fatalf("orientation/aspect: %+v", variant)
				}
				file, err := os.Open(filepath.Join(root, variant.Key))
				if err != nil {
					t.Fatal(err)
				}
				params := vips.NewImportParams()
				params.Access.Set(vips.AccessSequential)
				decoded, err := vips.LoadImageFromReader(file, params)
				if err != nil {
					file.Close()
					t.Fatal(err)
				}
				if decoded.Width() != int(variant.Width) || decoded.Height() != int(variant.Height) {
					t.Fatal("encoded dimensions differ")
				}
				for _, field := range decoded.GetFields() {
					if strings.HasPrefix(strings.ToLower(field), "exif") {
						t.Errorf("EXIF retained: %s", field)
					}
				}
				decoded.Close()
				file.Close()
				info, err := os.Stat(filepath.Join(root, variant.Key))
				if err != nil || info.Size() != variant.Size {
					t.Fatalf("stored size: %v", err)
				}
			}
			entries, err := os.ReadDir(filepath.Join(root, "final/sample"))
			if err != nil || len(entries) != 3 {
				t.Fatalf("files=%d error=%v", len(entries), err)
			}
		})
	}
}
