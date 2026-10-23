package httpx

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/share"
)

func TestShareManagementContract(t *testing.T) {
	h, store, pool, user := integrationUpload(t)
	ctx := context.Background()
	q := db.New(pool)
	file, err := q.CreateFile(ctx, db.CreateFileParams{OwnerID: user.ID, OriginalName: "ready.webp", Mime: "image/webp", Size: 7, StorageKey: "incoming/fixture"})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range [][2]string{{"uploaded", "scanning"}, {"scanning", "processing"}} {
		if _, err := q.AdvanceFile(ctx, db.AdvanceFileParams{ID: file.ID, ExpectedStatus: step[0], NextStatus: step[1]}); err != nil {
			t.Fatal(err)
		}
	}
	key := "final/" + file.ID.String() + "/1200.webp"
	if err := store.Put(ctx, key, strings.NewReader("encoded"), "image/webp"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: file.ID, StorageKey: key, Mime: "image/webp", Size: 7}); err != nil {
		t.Fatal(err)
	}
	service := &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store}
	shares := NewShareHandler(ShareHandler{Auth: h.Auth, Queries: q, Logger: h.Logger, Service: service}, nil)
	router := NewRouter(h.Logger, nil, nil, WithShares(shares))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+uploadTestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	path := "/v1/files/" + file.ID.String()
	settings := `{"expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","password":"secret","max_downloads":1}`
	w := request("POST", path+"/share", settings)
	var created share.Created
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &created) != nil || created.URL == "" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	w = request("GET", path+"/shares", "")
	var listing struct {
		Items []struct {
			ID pgtype.UUID `json:"id"`
		} `json:"items"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &listing) != nil || len(listing.Items) != 1 {
		t.Fatal(w.Body)
	}
	if strings.Contains(w.Body.String(), "token_hash") || strings.Contains(w.Body.String(), "password_hash") {
		t.Fatal("secrets leaked")
	}
	foreign, err := q.CreateUser(ctx, db.CreateUserParams{Email: file.ID.String() + "@share-foreign.test", PasswordHash: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", foreign.ID) })
	shares.Auth, err = auth.NewStub(foreign.ID.String(), uploadTestToken)
	if err != nil {
		t.Fatal(err)
	}
	if w = request("GET", path+"/shares", ""); w.Code != 404 {
		t.Fatal(w.Body)
	}
	revoke := "/v1/shares/" + listing.Items[0].ID.String()
	if w = request("DELETE", revoke, ""); w.Code != 404 {
		t.Fatal(w.Body)
	}
	shares.Auth = h.Auth
	if w = request("GET", created.URL, ""); w.Code != 401 {
		t.Fatal(w.Body)
	}
	r := httptest.NewRequest("GET", created.URL, nil)
	r.SetBasicAuth("", "secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "encoded" {
		t.Fatal(w.Body)
	}
	if w = request("GET", created.URL, ""); w.Code != 410 {
		t.Fatal(w.Body)
	}
	if w = request("DELETE", revoke, ""); w.Code != 204 {
		t.Fatal(w.Body)
	}
	if w = request("GET", created.URL, ""); w.Code != 410 {
		t.Fatal(w.Body)
	}
	if w = request("DELETE", revoke, ""); w.Code != 404 {
		t.Fatal(w.Body)
	}
}
