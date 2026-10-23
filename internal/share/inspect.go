package share

import (
	"context"
	"errors"
	"time"

	"safedrop/internal/db"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Info struct {
	Name      string    `json:"name"`
	MIME      string    `json:"mime"`
	Size      int64     `json:"size"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Inspect checks access without opening storage or reserving a download.
func (s *Service) Inspect(ctx context.Context, token, password string) (Info, error) {
	link, file, _, err := s.authorize(ctx, token, password)
	if errors.Is(err, ErrGone) && !errors.Is(err, ErrExpired) && !errors.Is(err, ErrLimitReached) {
		err = ErrNotFound
	}
	if err != nil {
		return Info{}, err
	}
	return Info{Name: file.OriginalName, MIME: file.Mime, Size: file.Size, ExpiresAt: link.ExpiresAt.Time}, nil
}

func (s *Service) authorize(ctx context.Context, token, password string) (db.ShareLink, db.File, []byte, error) {
	var link db.ShareLink
	var file db.File
	hash, err := tokenHash(token)
	if err != nil {
		return link, file, nil, err
	}
	link, err = s.Repository.Lookup(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrGone
	}
	if err != nil {
		return link, file, nil, err
	}
	if !link.ExpiresAt.Valid || !link.ExpiresAt.Time.After(time.Now()) {
		return link, file, nil, ErrExpired
	}
	if link.MaxDownloads.Valid && link.DownloadCount >= link.MaxDownloads.Int64 {
		return link, file, nil, ErrLimitReached
	}
	if link.PasswordHash.Valid && (len(password) > 72 || bcrypt.CompareHashAndPassword([]byte(link.PasswordHash.String), []byte(password)) != nil) {
		return link, file, nil, ErrPassword
	}
	file, err = s.Repository.File(ctx, link.FileID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrGone
	}
	if err == nil && (file.Status != "ready" || file.DeletedAt.Valid) {
		err = ErrGone
	}
	return link, file, hash, err
}
