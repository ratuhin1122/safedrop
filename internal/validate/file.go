package validate

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
)

// File is a rewound, disk-backed stream of the original bytes. Configuration
// checks do not establish that the compressed pixel data is valid or safe.
type File struct {
	MIME          string
	Size          int64
	Width, Height int
	file          *os.File
	once          sync.Once
	closeErr      error
}

func (f *File) Read(p []byte) (int, error)                   { return f.file.Read(p) }
func (f *File) Seek(offset int64, whence int) (int64, error) { return f.file.Seek(offset, whence) }

// Close releases the file and removes the temporary object, once.
func (f *File) Close() error {
	f.once.Do(func() { f.closeErr = errors.Join(f.file.Close(), os.Remove(f.file.Name())) })
	return f.closeErr
}

type requestReader struct {
	ctx  context.Context
	body io.Reader
}

func (r requestReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.body.Read(p)
}
