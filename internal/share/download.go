package share

import (
	"context"
	"errors"
	"io"
	"time"

	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
)

type Download struct {
	Body            io.ReadCloser
	URL, Name, MIME string
	Size            int64
}

func (s *Service) Open(ctx context.Context, token, password string) (Download, error) {
	link, file, hash, err := s.authorize(ctx, token, password)
	if err != nil {
		return Download{}, err
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
			return Download{}, s.unavailable(ctx, hash)
		}
		return Download{}, err
	}
	return download, nil
}

func (s *Service) unavailable(ctx context.Context, hash []byte) error {
	link, err := s.Repository.Lookup(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGone
	}
	if err != nil {
		return err
	}
	if !link.ExpiresAt.Valid || !link.ExpiresAt.Time.After(time.Now()) {
		return ErrExpired
	}
	if link.MaxDownloads.Valid && link.DownloadCount >= link.MaxDownloads.Int64 {
		return ErrLimitReached
	}
	return ErrGone
}
