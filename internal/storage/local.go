package storage

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/url"
	"os"
	"path"
)

type LocalConfig struct {
	Root string
	// Optional HTTP(S) URL at which this Local's ServeHTTP handler is mounted.
	BaseURL string
	// At least 32 bytes; required when BaseURL is set. Keep this secret.
	SigningKey []byte
}

type Local struct {
	root       *os.Root
	baseURL    *url.URL
	signingKey []byte
}

var _ Store = (*Local)(nil)

func NewLocal(cfg LocalConfig) (*Local, error) {
	if cfg.Root == "" {
		return nil, errors.New("storage root is required")
	}
	var base *url.URL
	if cfg.BaseURL != "" {
		var err error
		base, err = url.Parse(cfg.BaseURL)
		if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
			return nil, errors.New("invalid local download base URL")
		}
		if len(cfg.SigningKey) < 32 {
			return nil, errors.New("local signing key needs at least 32 bytes")
		}
		if base.Path == "" {
			base.Path = "/"
		}
	}
	if err := os.MkdirAll(cfg.Root, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(cfg.Root)
	if err != nil {
		return nil, err
	}
	for _, prefix := range []string{"incoming", "final"} {
		if err := root.MkdirAll(prefix, 0700); err != nil {
			root.Close()
			return nil, err
		}
	}
	return &Local{root: root, baseURL: base, signingKey: append([]byte(nil), cfg.SigningKey...)}, nil
}

// Close releases the root directory handle after all requests have stopped.
func (l *Local) Close() error { return l.root.Close() }

func (l *Local) Put(ctx context.Context, key string, source io.Reader, _ string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if source == nil {
		return errors.New("upload reader is required")
	}
	if err := l.root.MkdirAll(path.Dir(key), 0700); err != nil {
		return err
	}
	temp := path.Join(path.Dir(key), ".upload-"+rand.Text())
	file, err := l.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer l.root.Remove(temp)
	_, copyErr := io.Copy(file, contextReader{ctx, source})
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr, ctx.Err()); err != nil {
		return err
	}
	return l.root.Rename(temp, key)
}
