package httpx

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"safedrop/internal/db"
	"safedrop/internal/observability"
	"safedrop/internal/storage"
)

func TestUploadSafetyPostgres(t *testing.T) {
	png := uploadPNG(t)
	bomb := append([]byte(nil), png...)
	binary.BigEndian.PutUint32(bomb[16:20], 100000)
	binary.BigEndian.PutUint32(bomb[20:24], 100000)
	binary.BigEndian.PutUint32(bomb[29:33], crc32.ChecksumIEEE(bomb[12:29]))
	eicar := strings.Join([]string{"X5O!P%@AP[4", "\\PZX54(P^)7CC)7}$", "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"}, "")
	for _, tt := range []struct {
		name    string
		data    []byte
		status  int
		chunked bool
	}{
		{"image bomb", bomb, 400, false},
		{"PNG renamed to jpg", png, 201, false},
		{"raw EICAR fails image allowlist", []byte(eicar), 415, false},
		{"oversize declared length", append(append([]byte(nil), png...), make([]byte, 4096)...), 413, false},
		{"oversize streamed body", append(append([]byte(nil), png...), make([]byte, 4096)...), 413, true},
		{"tiny declared length", []byte{0x89}, 400, false},
		{"tiny streamed body", []byte{0x89}, 400, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _, pool, user := integrationUpload(t)
			h.Limits.MinBytes = 2
			h.Limits.MaxPixels = 20000000
			h.Metrics = observability.NewMetrics()
			root := t.TempDir()
			store, err := storage.NewLocal(storage.LocalConfig{Root: root})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			h.Storage = store
			server := httptest.NewServer(NewRouter(h.Logger, nil, nil, WithUploads(h)))
			defer server.Close()
			r := uploadRequest(tt.data)
			r.Header.Set("X-Filename", "renamed.jpg")
			r.Header.Set("Content-Type", "image/jpeg")
			if tt.chunked {
				r.ContentLength = -1
			}
			r.URL.Scheme, r.URL.Host, r.RequestURI = "http", strings.TrimPrefix(server.URL, "http://"), ""
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.status {
				t.Fatalf("status=%d want=%d", response.StatusCode, tt.status)
			}
			if tt.name == "image bomb" {
				if testutil.ToFloat64(h.Metrics.Uploads.WithLabelValues("rejected_pixels")) != 1 {
					t.Fatal("bomb was not rejected for its declared pixel count")
				}
			}
			want := 0
			if tt.status == 201 {
				want = 1
				var result struct {
					ID pgtype.UUID `json:"id"`
				}
				if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				file, err := db.New(pool).GetFile(context.Background(), result.ID)
				if err != nil {
					t.Fatal(err)
				}
				if file.Mime != "image/png" || file.OriginalName != "renamed.jpg" || file.Status != "uploaded" {
					t.Fatalf("file=%+v", file)
				}
				body, err := store.Get(context.Background(), file.StorageKey)
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(body)
				body.Close()
				if err != nil || string(got) != string(png) {
					t.Fatal("stored PNG changed")
				}
			}
			assertUploadCounts(t, pool, user, want)
			objects := 0
			if err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
				if err == nil && !entry.IsDir() {
					objects++
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if objects != want {
				t.Fatalf("objects=%d want=%d", objects, want)
			}
			entries, err := os.ReadDir(h.Limits.TempDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("validation temp files leaked: %v", err)
			}
		})
	}
}
