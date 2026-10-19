package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"time"
)

var (
	ErrInvalidKey      = errors.New("invalid storage key")
	ErrNotFound        = fs.ErrNotExist
	ErrPresignDisabled = errors.New("local download signing is not configured")
)

// Store never takes ownership of Put's reader; callers must close Get's result.
// Put may replace an existing key. Generate unique keys for independent uploads.
// Move deletes its source only after the destination is complete. S3 moves are
// not atomic: a failed delete can leave both objects, so retry the operation.
type Store interface {
	Put(context.Context, string, io.Reader, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Move(context.Context, string, string) error
	PresignGet(context.Context, string, time.Duration) (string, error)
}

func validateKey(key string) error {
	if !fs.ValidPath(key) || strings.ContainsAny(key, "\\:\x00") {
		return ErrInvalidKey
	}
	prefix, name, ok := strings.Cut(key, "/")
	if !ok || name == "" || (prefix != "incoming" && prefix != "final") {
		return ErrInvalidKey
	}
	return nil
}

func validateDownload(key string, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if !strings.HasPrefix(key, "final/") {
		return ErrInvalidKey
	}
	if ttl < time.Second || ttl > 7*24*time.Hour {
		return errors.New("download TTL must be between one second and seven days")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
