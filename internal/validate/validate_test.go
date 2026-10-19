package validate

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestValidate(t *testing.T) {
	pngData, jpegData := pngFixture(t), jpegFixture(t)
	for _, tt := range []struct {
		name, filename, clientType, wantMIME string
		data                                 []byte
		min, max                             int64
		pixels                               uint64
		wantErr                              error
	}{
		{"png", "image.png", "image/png", "image/png", pngData, 1, 4096, 100, nil},
		{"jpeg", "image.jpg", "image/jpeg", "image/jpeg", jpegData, 1, 4096, 100, nil},
		{"fake extension", "payload.exe", "text/plain", "image/png", pngData, 1, 4096, 100, nil},
		{"fake image", "image.png", "image/png", "", []byte("not an image"), 1, 4096, 100, ErrType},
		{"webp extended", "image.bin", "text/plain", "image/webp", webpFixture("VP8X", 2, 3), 1, 4096, 100, nil},
		{"webp lossless", "image.bin", "", "image/webp", webpFixture("VP8L", 2, 3), 1, 4096, 100, nil},
		{"webp lossy", "image.bin", "", "image/webp", webpFixture("VP8 ", 2, 3), 1, 4096, 100, nil},
		{"heic", "image.png", "image/png", "image/heic", heicFixture(2, 3), 1, 4096, 100, nil},
		{"empty", "image.png", "", "", nil, 1, 4096, 100, ErrTooSmall},
		{"below min", "", "", "", pngData, int64(len(pngData) + 1), 4096, 100, ErrTooSmall},
		{"exact bounds", "", "", "image/png", pngData, int64(len(pngData)), int64(len(pngData)), 6, nil},
		{"above max", "", "", "", pngData, 1, int64(len(pngData) - 1), 100, ErrTooLarge},
		{"oversized dimensions", "image.png", "", "", hugePNG(t), 1, 4096, 20000000, ErrPixels},
		{"pixel boundary exceeded", "", "", "", pngData, 1, 4096, 5, ErrPixels},
		{"heic bomb", "", "", "", heicFixture(100000, 100000), 1, 4096, 20000000, ErrPixels},
		{"webp bomb", "", "", "", webpFixture("VP8X", 10000, 10000), 1, 4096, 20000000, ErrPixels},
		{"truncated png", "", "", "", pngData[:24], 1, 4096, 100, ErrImage},
		{"gif disallowed", "image.png", "image/png", "", []byte("GIF89a123456789"), 1, 4096, 100, ErrType},
		{"truncated heic", "", "", "", heicFixture(2, 3)[:30], 1, 4096, 100, ErrImage},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			r := httptest.NewRequest(http.MethodPost, "/uploads?filename="+tt.filename, bytes.NewReader(tt.data))
			r.Header.Set("Content-Type", tt.clientType)
			// A dishonest Content-Length must not bypass byte counting.
			r.ContentLength = 1
			file, err := Validate(httptest.NewRecorder(), r, Limits{tt.min, tt.max, tt.pixels, dir})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if file != nil {
					file.Close()
					t.Fatal("returned file on failure")
				}
				if errors.Is(err, ErrTooLarge) {
					var maxErr *http.MaxBytesError
					if !errors.As(err, &maxErr) {
						t.Fatal("missing MaxBytesReader error")
					}
				}
			} else {
				if file.MIME != tt.wantMIME || file.Size != int64(len(tt.data)) || file.Width != 2 || file.Height != 3 {
					t.Fatalf("metadata: %+v", file)
				}
				copied, err := io.ReadAll(file)
				if err != nil || !bytes.Equal(copied, tt.data) {
					t.Fatalf("original stream changed: %v", err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatalf("second Close: %v", err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("temporary files leaked: %v %v", entries, err)
			}
		})
	}
}
