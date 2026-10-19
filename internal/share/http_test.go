package share_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

func TestDownloadRules(t *testing.T) {
	for _, tt := range []struct {
		name, password string
		status         int
	}{
		{"expired", "", 410}, {"over limit", "", 410}, {"invalid token", "", 410},
		{"wrong password", "wrong", 401}, {"correct password", "secret", 200}, {"normal", "", 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service, repo, token, hash := fixture(t)
			link := repo.links[string(hash)]
			switch tt.name {
			case "expired":
				link.ExpiresAt.Time = time.Now().Add(-time.Minute)
			case "over limit":
				link.MaxDownloads = pgtype.Int8{Int64: 1, Valid: true}
				link.DownloadCount = 1
			case "invalid token":
				token = "invalid"
			case "wrong password", "correct password":
				password, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
				link.PasswordHash = pgtype.Text{String: string(password), Valid: true}
			}
			repo.links[string(hash)] = link
			var logs bytes.Buffer
			r := httptest.NewRequest(http.MethodGet, "/s/"+token, nil)
			if tt.password != "" {
				r.SetBasicAuth("", tt.password)
			}
			w := httptest.NewRecorder()
			router(service, &logs).ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if strings.Contains(logs.String(), token) {
				t.Fatal("token leaked into request logs")
			}
			if tt.status == 200 && w.Body.String() != "safe image" {
				t.Fatalf("body=%q", w.Body.String())
			}
			current, _ := repo.Lookup(context.Background(), hash)
			if tt.status != 200 && current.DownloadCount != link.DownloadCount {
				t.Fatal("rejected request consumed a download")
			}
		})
	}
}

func TestConcurrentDownloadLimit(t *testing.T) {
	service, repo, token, hash := fixture(t)
	link := repo.links[string(hash)]
	link.MaxDownloads = pgtype.Int8{Int64: 1, Valid: true}
	repo.links[string(hash)] = link
	handler := router(service, io.Discard)
	var successes, gone atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/s/"+token, nil))
			if w.Code == 200 {
				successes.Add(1)
			} else if w.Code == 410 {
				gone.Add(1)
			}
		}()
	}
	group.Wait()
	if successes.Load() != 1 || gone.Load() != 7 {
		t.Fatalf("success=%d gone=%d", successes.Load(), gone.Load())
	}
}
