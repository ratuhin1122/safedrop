package httpx

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/storage"
	"safedrop/internal/validate"
)

const uploadTestUser = "11111111-1111-4111-8111-111111111111"

var uploadTestToken = strings.Repeat("t", 32)

type uploadRepositoryFunc func(context.Context, db.CreateFileParams) (db.File, error)

func (f uploadRepositoryFunc) CreateUpload(ctx context.Context, p db.CreateFileParams) (db.File, error) {
	return f(ctx, p)
}

func uploadPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func testUploadHandler(t *testing.T) (*UploadHandler, *storage.Local) {
	t.Helper()
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	authenticator, err := auth.NewStub(uploadTestUser, uploadTestToken)
	if err != nil {
		t.Fatal(err)
	}
	return &UploadHandler{Auth: authenticator, Storage: store,
		Limits: validate.Limits{MinBytes: 1, MaxBytes: 4096, MaxPixels: 100, TempDir: t.TempDir()},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		Repository: uploadRepositoryFunc(func(context.Context, db.CreateFileParams) (db.File, error) {
			t.Error("repository unexpectedly called")
			return db.File{}, nil
		})}, store
}

func uploadRequest(data []byte) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/files", bytes.NewReader(data))
	r.Header.Set("Authorization", "Bearer "+uploadTestToken)
	r.Header.Set("X-Filename", "image.exe")
	r.Header.Set("Content-Type", "text/plain")
	return r
}
