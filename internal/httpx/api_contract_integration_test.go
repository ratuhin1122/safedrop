package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
)

func TestAPIContractPostgres(t *testing.T) {
	h, store, pool, user := integrationUpload(t)
	q := db.New(pool)
	files := &FilesHandler{Auth: h.Auth, Queries: q, Storage: store}
	router := NewRouter(h.Logger, nil, nil, WithUploads(h), WithFiles(files), WithUsage(files, q, 12345, 20))
	request := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("Authorization", "Bearer "+uploadTestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, multipartRequest(t, uploadPNG(t), "renamed.jpg", 1))
	var result struct {
		ID     pgtype.UUID `json:"id"`
		Status string      `json:"status"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Status != "uploaded" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	assertUploadCounts(t, pool, user, 1)
	ctx := context.Background()
	for _, step := range [][2]string{{"uploaded", "scanning"}, {"scanning", "processing"}} {
		if _, err := q.AdvanceFile(ctx, db.AdvanceFileParams{ID: result.ID, ExpectedStatus: step[0], NextStatus: step[1]}); err != nil {
			t.Fatal(err)
		}
	}
	for _, width := range []int32{375, 768, 1200} {
		key := fmt.Sprintf("final/%s/%d.webp", result.ID.String(), width)
		if err := store.Put(ctx, key, strings.NewReader("encoded"), "image/webp"); err != nil {
			t.Fatal(err)
		}
		if err := q.UpsertFileVariant(ctx, db.UpsertFileVariantParams{FileID: result.ID, TargetWidth: width, Width: width, Height: 200, Size: 7, Mime: "image/webp", StorageKey: key}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: result.ID, StorageKey: "final/" + result.ID.String() + "/1200.webp", Mime: "image/webp", Size: 7}); err != nil {
		t.Fatal(err)
	}
	w = request("GET", "/v1/files")
	var page struct {
		Items []fileView `json:"items"`
		Next  string     `json:"next_cursor"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != 1 {
		t.Fatal(w.Body)
	}
	for _, width := range []string{"375", "768", "1200"} {
		v := page.Items[0].Variants["w"+width]
		if v == nil || v.Size != 7 || v.MIME != "image/webp" || v.URL == "" {
			t.Fatalf("variant=%+v", v)
		}
		if download := request("GET", v.URL); download.Code != 200 || download.Body.String() != "encoded" {
			t.Fatal(download.Body)
		}
	}
	assertUsageContract(t, request("GET", "/v1/me/usage"), int64(len(uploadPNG(t))))
	foreign, err := q.CreateUser(ctx, db.CreateUserParams{Email: result.ID.String() + "@foreign.test", PasswordHash: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "UPDATE files SET owner_id=$1 WHERE id=$2", user.ID, result.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", foreign.ID)
	})
	_, err = pool.Exec(ctx, "UPDATE files SET owner_id=$1 WHERE id=$2", foreign.ID, result.ID)
	if err != nil {
		t.Fatal(err)
	}
	if denied := request("GET", page.Items[0].Variants["w375"].URL); denied.Code != http.StatusNotFound {
		t.Fatal(denied.Code)
	}
	if _, err := pool.Exec(ctx, "UPDATE files SET owner_id=$1 WHERE id=$2", user.ID, result.ID); err != nil {
		t.Fatal(err)
	}
	if deleted := request("DELETE", "/v1/files/"+result.ID.String()); deleted.Code != 204 {
		t.Fatal(deleted.Body)
	}
	assertUsageContract(t, request("GET", "/v1/me/usage"), int64(len(uploadPNG(t))))
}

func assertUsageContract(t *testing.T, w *httptest.ResponseRecorder, wantBytes int64) {
	t.Helper()
	var value struct {
		BytesToday int64 `json:"bytes_today"`
		BytesLimit int64 `json:"bytes_limit"`
		Uploads    int   `json:"uploads_this_hour"`
		Limit      int   `json:"uploads_limit"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &value) != nil {
		t.Fatal(w.Body)
	}
	if value.BytesToday != wantBytes || value.BytesLimit != 12345 || value.Uploads != 1 || value.Limit != 20 {
		t.Fatalf("usage=%+v", value)
	}
}
