package share_test

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http/httptest"
	"safedrop/internal/share"
	"testing"
	"time"
)

func TestPublicInfoScreens(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
	}{{"ready", 200}, {"password", 401}, {"missing", 404}, {"expired", 410}, {"limit", 410}} {
		t.Run(tc.name, func(t *testing.T) {
			service, repo, token, hash := fixture(t)
			link := repo.links[string(hash)]
			switch tc.name {
			case "password":
				hashed, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
				link.PasswordHash = pgtype.Text{String: string(hashed), Valid: true}
			case "missing":
				delete(repo.links, string(hash))
			case "expired":
				link.ExpiresAt.Time = time.Now().Add(-time.Hour)
			case "limit":
				link.MaxDownloads = pgtype.Int8{Int64: 1, Valid: true}
				link.DownloadCount = 1
			}
			if tc.name != "missing" {
				repo.links[string(hash)] = link
			}
			handler := router(service, io.Discard)
			for i := 0; i < 2; i++ {
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, httptest.NewRequest("GET", "/s/"+token+"?info=1", nil))
				if res.Code != tc.status {
					t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
				}
				if res.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatal("missing noindex")
				}
				if tc.status == 200 {
					var info share.Info
					if err := json.Unmarshal(res.Body.Bytes(), &info); err != nil {
						t.Fatal(err)
					}
					if info.Name != "image.jpg" || info.Size != 10 || info.ExpiresAt.IsZero() {
						t.Fatalf("info=%+v", info)
					}
				}
			}
			if tc.name == "ready" && repo.links[string(hash)].DownloadCount != 0 {
				t.Fatal("metadata consumed a download")
			}
			if tc.name == "password" {
				req := httptest.NewRequest("GET", "/s/"+token+"?info=1", nil)
				req.SetBasicAuth("", "secret")
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				if res.Code != 200 {
					t.Fatalf("unlocked status=%d", res.Code)
				}
			}
		})
	}
}

func TestInspectDoesNotOpenStorage(t *testing.T) {
	service, repo, token, hash := fixture(t)
	service.Storage = nil
	if _, err := service.Inspect(context.Background(), token, ""); err != nil {
		t.Fatal(err)
	}
	if repo.links[string(hash)].DownloadCount != 0 {
		t.Fatal("count changed")
	}
}
