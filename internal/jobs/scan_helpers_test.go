package jobs

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

type scannerFunc func(context.Context, io.Reader) (scan.Result, error)

func (f scannerFunc) Scan(ctx context.Context, reader io.Reader) (scan.Result, error) {
	return f(ctx, reader)
}

// Keep EICAR in memory: a host antivirus may quarantine on-disk test payloads.
type scanStore struct {
	storage.Store
	data                 []byte
	deleted              bool
	deletes, failDeletes int
	reads                int
}

func (s *scanStore) Get(context.Context, string) (io.ReadCloser, error) {
	s.reads++
	if s.deleted {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(s.data)), nil
}
func (s *scanStore) Delete(context.Context, string) error {
	s.deletes++
	if s.failDeletes > 0 {
		s.failDeletes--
		return errors.New("storage temporarily unavailable")
	}
	s.deleted = true
	return nil
}

func scanFixtures(t *testing.T) ([]byte, []byte) {
	t.Helper()
	eicar := strings.Join([]string{"X5O!P%@AP[4", "\\PZX54(P^)7CC)7}$", "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"}, "")
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return []byte(eicar), data.Bytes()
}
