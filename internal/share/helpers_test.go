package share_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/httpx"
	"safedrop/internal/share"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type memoryRepository struct {
	mu      sync.Mutex
	file    db.File
	links   map[string]db.ShareLink
	created db.CreateShareLinkParams
}

func (m *memoryRepository) Create(_ context.Context, p db.CreateShareLinkParams) (db.ShareLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.OwnerID != m.file.OwnerID || p.FileID != m.file.ID || m.file.Status != "ready" {
		return db.ShareLink{}, pgx.ErrNoRows
	}
	m.created = p
	row := db.ShareLink{TokenHash: p.TokenHash, FileID: p.FileID, ExpiresAt: p.ExpiresAt, PasswordHash: p.PasswordHash, MaxDownloads: p.MaxDownloads}
	m.links[string(p.TokenHash)] = row
	return row, nil
}
func (m *memoryRepository) Lookup(_ context.Context, hash []byte) (db.ShareLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.links[string(hash)]
	if !ok {
		return row, pgx.ErrNoRows
	}
	return row, nil
}
func (m *memoryRepository) File(context.Context, pgtype.UUID) (db.File, error) { return m.file, nil }
func (m *memoryRepository) Consume(_ context.Context, hash []byte) (db.ShareLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.links[string(hash)]
	if !ok || !row.ExpiresAt.Time.After(time.Now()) || (row.MaxDownloads.Valid && row.DownloadCount >= row.MaxDownloads.Int64) {
		return row, pgx.ErrNoRows
	}
	row.DownloadCount++
	m.links[string(hash)] = row
	return row, nil
}

func fixture(t *testing.T) (*share.Service, *memoryRepository, string, []byte) {
	t.Helper()
	var owner, id pgtype.UUID
	_ = owner.Scan("11111111-1111-4111-8111-111111111111")
	_ = id.Scan("22222222-2222-4222-8222-222222222222")
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Put(context.Background(), "final/test/1200.webp", strings.NewReader("safe image"), "image/webp"); err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepository{file: db.File{ID: id, OwnerID: owner, Status: "ready", StorageKey: "final/test/1200.webp", OriginalName: "image.jpg", Mime: "image/webp", Size: 10}, links: map[string]db.ShareLink{}}
	raw := []byte("0123456789abcdef")
	hash := sha256.Sum256(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	repo.links[string(hash[:])] = db.ShareLink{TokenHash: hash[:], FileID: id, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}}
	return &share.Service{Repository: repo, Storage: store}, repo, token, hash[:]
}

func router(service *share.Service, output io.Writer) http.Handler {
	authenticator, _ := auth.NewStub("11111111-1111-4111-8111-111111111111", strings.Repeat("t", 32))
	logger := slog.New(slog.NewJSONHandler(output, nil))
	return httpx.NewRouter(logger, nil, nil, httpx.WithShares(&httpx.ShareHandler{Auth: authenticator, Service: service, Logger: logger}))
}
