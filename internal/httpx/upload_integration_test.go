package httpx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUploadHTTPCommitsFileAndRiverJob(t *testing.T) {
	h, store, pool, user := integrationUpload(t)
	server := httptest.NewServer(NewRouter(h.Logger, nil, nil, WithUploads(h)))
	defer server.Close()
	data := uploadPNG(t)
	r := uploadRequest(data)
	r.URL.Scheme, r.URL.Host = "http", strings.TrimPrefix(server.URL, "http://")
	r.RequestURI = ""
	response, err := server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		ID string `json:"id"`
	}
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("upload status=%d body=%s", response.StatusCode, body)
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	var id pgtype.UUID
	if err := id.Scan(result.ID); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	file, err := db.New(pool).GetFile(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if file.Status != "uploaded" || file.OwnerID != user.ID || file.Size != int64(len(data)) || file.Mime != "image/png" || !strings.HasPrefix(file.StorageKey, "incoming/") {
		t.Fatalf("file row: %+v", file)
	}
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='process_file' AND args->>'file_id'=$1", result.ID).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("River job count=%d error=%v", count, err)
	}
	body, err := store.Get(ctx, file.StorageKey)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	stored, err := io.ReadAll(body)
	if err != nil || string(stored) != string(data) {
		t.Fatalf("stored bytes: %v", err)
	}
}
