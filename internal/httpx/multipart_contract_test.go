package httpx

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"safedrop/internal/db"
)

func multipartRequest(t *testing.T, data []byte, filename string, count int) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for i := 0; i < count; i++ {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/files", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Authorization", "Bearer "+uploadTestToken)
	return r
}

func TestMultipartContract(t *testing.T) {
	png := uploadPNG(t)
	bomb := append([]byte(nil), png...)
	binary.BigEndian.PutUint32(bomb[16:20], 100000)
	binary.BigEndian.PutUint32(bomb[20:24], 100000)
	binary.BigEndian.PutUint32(bomb[29:33], crc32.ChecksumIEEE(bomb[12:29]))
	for _, tt := range []struct {
		name          string
		data          []byte
		filename      string
		count, status int
		code          string
	}{
		{"renamed PNG", png, "renamed.jpg", 1, 201, ""},
		{"oversize", append(png, make([]byte, 4096)...), "x.png", 1, 413, "too_large"},
		{"pixels", bomb, "x.png", 1, 422, "pixel_limit"},
		{"bad type", []byte("not an image"), "x.png", 1, 415, "bad_type"},
		{"path", png, "../x.png", 1, 400, "invalid_request"},
		{"multiple files", png, "x.png", 2, 400, "invalid_request"},
		{"empty form", nil, "x.png", 0, 400, "invalid_request"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := testUploadHandler(t)
			created := 0
			h.Repository = uploadRepositoryFunc(func(_ context.Context, p db.CreateFileParams) (db.File, error) {
				created++
				if p.OriginalName != "renamed.jpg" || p.Mime != "image/png" {
					t.Fatalf("params=%+v", p)
				}
				return db.File{ID: p.OwnerID}, nil
			})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, multipartRequest(t, tt.data, tt.filename, tt.count))
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			var payload map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if tt.status == 201 {
				if created != 1 || payload["status"] != "uploaded" || payload["id"] == nil {
					t.Fatal(payload)
				}
			} else if created != 0 || payload["code"] != tt.code || payload["message"] == nil {
				t.Fatal(payload)
			}
		})
	}
}

func TestJSONRoutingErrors(t *testing.T) {
	h, _ := testUploadHandler(t)
	router := NewRouter(h.Logger, nil, nil, WithUploads(h))
	for _, path := range []string{"/v1/missing", "/v1/files"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal(w.Header())
		}
		var result struct{ Code, Message string }
		if json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Code == "" || result.Message == "" {
			t.Fatal(w.Body)
		}
	}
}
