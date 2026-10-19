package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
)

func (l *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := l.root.Open(key)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("storage object is not a regular file")
	}
	return &localReader{contextReader{ctx, file}, file}, nil
}

type localReader struct {
	contextReader
	file *os.File
}

func (r *localReader) Close() error { return r.file.Close() }

func (l *Local) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := l.root.Remove(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) Move(ctx context.Context, source, target string) error {
	if err := validateKey(source); err != nil {
		return err
	}
	if err := validateKey(target); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Reject directories rather than moving an entire tree of objects.
	file, err := l.Get(ctx, source)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := l.root.MkdirAll(path.Dir(target), 0700); err != nil {
		return err
	}
	return l.root.Rename(source, target)
}
