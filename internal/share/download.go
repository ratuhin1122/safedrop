package share

import (
	"context"
	"errors"
	"io"
	"time"

	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Download struct {
	Body            io.ReadCloser
	URL, Name, MIME string
	Size            int64
}

func (s *Service) Open(ctx context.Context, token, password string) (Download, error) {
	hash, err := tokenHash(token)
	if err != nil {
		return Download{}, err
	}
	link, err := s.Repository.Lookup(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Download{}, ErrGone
	}
	if err != nil {
		return Download{}, err
	}
	if !link.ExpiresAt.Valid || !link.ExpiresAt.Time.After(time.Now()) ||
		(link.MaxDownloads.Valid && link.DownloadCount >= link.MaxDownloads.Int64) {
		return Download{}, ErrGone
	}
	if link.PasswordHash.Valid {
		if len(password) > 72 || bcrypt.CompareHashAndPassword([]byte(link.PasswordHash.String), []byte(password)) != nil {
			return Download{}, ErrPassword
		}
	}
	file, err := s.Repository.File(ctx, link.FileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Download{}, ErrGone
	}
	if err != nil {
		return Download{}, err
	}
	if file.Status != "ready" || file.DeletedAt.Valid {
		return Download{}, ErrGone
	}
	download := Download{Name: file.OriginalName, MIME: file.Mime, Size: file.Size}
	// Presigned URLs are reusable. Bounded-download links always stream through
	// this endpoint, keeping their count enforcement in the atomic SQL update.
	ttl := min(time.Minute, time.Until(link.ExpiresAt.Time))
	if s.Presign && !link.MaxDownloads.Valid && ttl >= time.Second {
		download.URL, err = s.Storage.PresignGet(ctx, file.StorageKey, ttl)
		if errors.Is(err, storage.ErrPresignDisabled) {
			err = nil
			download.URL = ""
		}
		if err != nil {
			return Download{}, err
		}
	}
	if download.URL == "" {
		download.Body, err = s.Storage.Get(ctx, file.StorageKey)
		if errors.Is(err, storage.ErrNotFound) {
			return Download{}, ErrGone
		}
		if err != nil {
			return Download{}, err
		}
	}
	if _, err := s.Repository.Consume(ctx, hash); err != nil {
		if download.Body != nil {
			download.Body.Close()
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Download{}, ErrGone
		}
		return Download{}, err
	}
	return download, nil
}
