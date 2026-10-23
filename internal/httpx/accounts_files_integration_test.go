package httpx

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"safedrop/internal/auth"
	"safedrop/internal/db"
)

func TestAccountsAndFileOwnership(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	q := db.New(pool)
	tokens, _ := auth.NewTokens(strings.Repeat("k", 32))
	accounts := auth.NewAccounts(q, tokens)
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil,
		WithAccounts(accounts), WithFiles(&FilesHandler{Auth: tokens, Queries: q}))
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	register := func() (string, db.User) {
		email := strings.ToLower(rand.Text()) + "@auth.test"
		body := `{"email":"` + email + `","password":"test password long enough"}`
		w := request("POST", "/v1/auth/register", "", body)
		if w.Code != 201 {
			t.Fatalf("register: %d %s", w.Code, w.Body)
		}
		var result struct {
			Token string `json:"token"`
		}
		if json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Token == "" {
			t.Fatal("missing token")
		}
		u, err := q.GetUserByEmail(ctx, email)
		if err != nil || !strings.HasPrefix(u.PasswordHash, "$argon2id$") {
			t.Fatalf("stored account: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, "DELETE FROM files WHERE owner_id=$1", u.ID)
			_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID)
		})
		if w := request("POST", "/v1/auth/login", "", body); w.Code != 200 {
			t.Fatalf("login: %d", w.Code)
		}
		if w := request("POST", "/v1/auth/register", "", body); w.Code != 409 {
			t.Fatalf("duplicate: %d", w.Code)
		}
		if w := request("POST", "/v1/auth/login", "", strings.Replace(body, "test password long enough", "incorrect password long", 1)); w.Code != 401 {
			t.Fatalf("wrong password: %d", w.Code)
		}
		return result.Token, u
	}
	aToken, a := register()
	bToken, b := register()
	makeFile := func(u db.User) db.File {
		f, err := q.CreateFile(ctx, db.CreateFileParams{OwnerID: u.ID, OriginalName: "a.png", Mime: "image/png", Size: 42, StorageKey: "incoming/" + rand.Text()})
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	first, second, foreign := makeFile(a), makeFile(a), makeFile(b)
	for _, token := range []string{aToken, bToken} {
		w := request("GET", "/v1/files?limit=100", token, "")
		var listing struct {
			Files []fileView `json:"items"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &listing) != nil {
			t.Fatal("list failed")
		}
		want := 2
		if token == bToken {
			want = 1
		}
		if len(listing.Files) != want {
			t.Fatal("unexpected owner file count")
		}
		for _, file := range listing.Files {
			if (token == aToken && file.ID == foreign.ID) || (token == bToken && file.ID != foreign.ID) {
				t.Fatal("foreign file listed")
			}
		}
	}
	if _, err := pool.Exec(ctx, "UPDATE files SET created_at='2026-01-01T00:00:00Z' WHERE owner_id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "DELETE"} {
		if w := request(method, "/v1/files/"+foreign.ID.String(), aToken, ""); w.Code != 404 {
			t.Fatalf("foreign %s: %d", method, w.Code)
		}
		if w := request(method, "/v1/files/"+first.ID.String(), "", ""); w.Code != 401 {
			t.Fatalf("unauthenticated: %d", w.Code)
		}
	}
	if w := request("GET", "/v1/files/"+foreign.ID.String(), bToken, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	var page struct {
		Files []fileView `json:"items"`
		Next  string     `json:"next_cursor"`
	}
	w := request("GET", "/v1/files?limit=1", aToken, "")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Files) != 1 || page.Next == "" {
		t.Fatalf("page: %s", w.Body)
	}
	seen := page.Files[0].ID
	w = request("GET", "/v1/files?limit=1&cursor="+page.Next, aToken, "")
	page.Next = ""
	if json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Files) != 1 || page.Files[0].ID == seen || page.Next != "" {
		t.Fatalf("next page: %s", w.Body)
	}
	if page.Files[0].ID != first.ID && page.Files[0].ID != second.ID {
		t.Fatal("foreign file listed")
	}
	if w := request("DELETE", "/v1/files/"+first.ID.String(), aToken, ""); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if w := request("GET", "/v1/files/"+first.ID.String(), aToken, ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	var deleted bool
	if err := pool.QueryRow(ctx, "SELECT deleted_at IS NOT NULL FROM files WHERE id=$1 AND owner_id=$2", first.ID, a.ID).Scan(&deleted); err != nil || !deleted {
		t.Fatalf("soft deletion: %v %v", deleted, err)
	}
}
