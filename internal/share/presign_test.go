package share_test

import (
	"context"
	"testing"
	"time"

	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
)

type presignStore struct {
	storage.Store
	calls int
	ttl   time.Duration
}

func (s *presignStore) PresignGet(_ context.Context, _ string, ttl time.Duration) (string, error) {
	s.calls++
	s.ttl = ttl
	return "https://storage.example.test/object?signed=true", nil
}

func TestPresignNeverBypassesDownloadCaps(t *testing.T) {
	for _, capped := range []bool{false, true} {
		service, repo, token, hash := fixture(t)
		store := &presignStore{Store: service.Storage}
		service.Storage, service.Presign = store, true
		link := repo.links[string(hash)]
		link.ExpiresAt.Time = time.Now().Add(30 * time.Second)
		if capped {
			link.MaxDownloads = pgtype.Int8{Int64: 1, Valid: true}
		}
		repo.links[string(hash)] = link
		download, err := service.Open(context.Background(), token, "")
		if err != nil {
			t.Fatal(err)
		}
		if capped {
			if download.Body == nil || download.URL != "" || store.calls != 0 {
				t.Fatal("capped link produced reusable presigned URL")
			}
			download.Body.Close()
		} else if download.URL == "" || store.calls != 1 || store.ttl > 30*time.Second || store.ttl <= 0 {
			t.Fatalf("presign calls=%d ttl=%v", store.calls, store.ttl)
		}
	}
}
