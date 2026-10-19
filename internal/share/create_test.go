package share_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"safedrop/internal/share"

	"golang.org/x/crypto/bcrypt"
)

func TestCreateStoresOnlyTokenAndPasswordHashes(t *testing.T) {
	service, repo, _, _ := fixture(t)
	maximum := int64(2)
	created, err := service.Create(context.Background(), repo.file.OwnerID, repo.file.ID,
		share.CreateOptions{ExpiresAt: time.Now().Add(time.Hour), Password: "secret", MaxDownloads: &maximum})
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(created.URL, "/s/")
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 16 {
		t.Fatalf("token length=%d error=%v", len(raw), err)
	}
	hash := sha256.Sum256(raw)
	if string(repo.created.TokenHash) != string(hash[:]) {
		t.Fatal("wrong persisted token hash")
	}
	if repo.created.PasswordHash.String == "secret" || bcrypt.CompareHashAndPassword([]byte(repo.created.PasswordHash.String), []byte("secret")) != nil {
		t.Fatal("password was not hashed")
	}
	if !repo.created.MaxDownloads.Valid || repo.created.MaxDownloads.Int64 != 2 {
		t.Fatal("download cap not persisted")
	}
}

func TestInvalidCreateSettings(t *testing.T) {
	service, repo, _, _ := fixture(t)
	zero := int64(0)
	for _, options := range []share.CreateOptions{
		{ExpiresAt: time.Now().Add(-time.Minute)},
		{ExpiresAt: time.Now().Add(time.Hour), MaxDownloads: &zero},
		{ExpiresAt: time.Now().Add(time.Hour), Password: strings.Repeat("x", 73)},
	} {
		if _, err := service.Create(context.Background(), repo.file.OwnerID, repo.file.ID, options); !errors.Is(err, share.ErrInvalid) {
			t.Fatalf("error=%v", err)
		}
	}
}
