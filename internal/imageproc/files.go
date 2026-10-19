package imageproc

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"safedrop/internal/observability"
	"safedrop/internal/storage"
)

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}

func spool(ctx context.Context, store storage.Store, key, output string, maxBytes int64) error {
	body, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		body.Close()
		return err
	}
	n, copyErr := io.Copy(file, io.LimitReader(contextReader{ctx, body}, maxBytes+1))
	err = errors.Join(copyErr, file.Close(), body.Close())
	if err != nil {
		return err
	}
	if n > maxBytes {
		return errors.New("image input exceeds maximum size")
	}
	return ctx.Err()
}

func upload(ctx context.Context, store storage.Store, key, path, mime string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if info.Size() < 1 {
		return 0, errors.New("encoder produced an empty file")
	}
	staged := "incoming/processed/" + strings.TrimPrefix(key, "final/")
	moved := false
	defer func() {
		if !moved {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = store.Delete(cleanup, staged)
		}
	}()
	if err := store.Put(ctx, staged, contextReader{ctx, file}, mime); err != nil {
		return 0, err
	}
	moveCtx, endMove := observability.Span(ctx, "storage.move")
	err = store.Move(moveCtx, staged, key)
	endMove(err)
	if err != nil {
		return 0, err
	}
	moved = true
	return info.Size(), nil
}
