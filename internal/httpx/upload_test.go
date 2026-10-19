package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUploadCreated(t *testing.T) {
	h, store := testUploadHandler(t)
	data := uploadPNG(t)
	const id = "22222222-2222-4222-8222-222222222222"
	var key string
	h.Repository = uploadRepositoryFunc(func(ctx context.Context, p db.CreateFileParams) (db.File, error) {
		if p.OwnerID.String() != uploadTestUser || p.Mime != "image/png" || p.OriginalName != "image.exe" || p.Size != int64(len(data)) || !strings.HasPrefix(p.StorageKey, "incoming/") {
			t.Fatalf("params: %+v", p)
		}
		key = p.StorageKey
		body, err := store.Get(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		body.Close()
		var fileID pgtype.UUID
		if err := fileID.Scan(id); err != nil {
			t.Fatal(err)
		}
		return db.File{ID: fileID, Status: "uploaded"}, nil
	})
	w := httptest.NewRecorder()
	NewRouter(h.Logger, nil, nil, WithUploads(h)).ServeHTTP(w, uploadRequest(data))
	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || response.ID != id {
		t.Fatalf("response: %d %s", w.Code, w.Body.String())
	}
	body, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil || string(got) != string(data) {
		t.Fatalf("stored bytes differ: %v", err)
	}
}

func TestUploadPersistenceFailureCleanup(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		h, store := testUploadHandler(t)
		var key string
		h.Repository = uploadRepositoryFunc(func(_ context.Context, p db.CreateFileParams) (db.File, error) {
			key = p.StorageKey
			if uncertain {
				return db.File{}, ErrUploadCommitUncertain
			}
			return db.File{}, errors.New("enqueue failed")
		})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, uploadRequest(uploadPNG(t)))
		if w.Code != 500 {
			t.Fatalf("status: %d", w.Code)
		}
		body, err := store.Get(context.Background(), key)
		if body != nil {
			body.Close()
		}
		if uncertain && err != nil {
			t.Fatalf("deleted object after uncertain commit: %v", err)
		}
		if !uncertain && !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("orphan object: %v", err)
		}
	}
}

func TestUploadInvalidContent(t *testing.T) {
	for _, tt := range []struct {
		name   string
		data   []byte
		status int
	}{
		{"unsupported type", []byte("not an image"), 415},
		{"invalid png", []byte("\x89PNG\r\n\x1a\ntruncated"), 400},
		{"actual size exceeds max", []byte(strings.Repeat("x", 4097)), 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := testUploadHandler(t)
			r := uploadRequest(tt.data)
			r.ContentLength = -1
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
