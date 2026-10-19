package validate

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
)

var (
	ErrTooSmall = errors.New("upload is smaller than the minimum")
	ErrTooLarge = errors.New("upload exceeds the maximum")
	ErrType     = errors.New("unsupported image type")
	ErrImage    = errors.New("invalid image configuration")
	ErrPixels   = errors.New("image exceeds the maximum pixel count")
)

type Limits struct {
	MinBytes, MaxBytes int64
	MaxPixels          uint64
	// Empty uses the operating system's temporary directory.
	TempDir string
}

// Validate consumes a request whose body is the raw file, not a multipart form.
// It ignores the filename, Content-Length, and client Content-Type. It replaces
// and closes r.Body. On success, the caller must Close the returned File.
func Validate(w http.ResponseWriter, r *http.Request, limits Limits) (*File, error) {
	if limits.MinBytes < 1 || limits.MaxBytes < limits.MinBytes || limits.MaxPixels == 0 {
		return nil, errors.New("positive size and pixel limits are required; min must not exceed max")
	}
	if r == nil || r.Body == nil {
		return nil, errors.New("upload body is required")
	}
	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxBytes)
	defer r.Body.Close()
	temp, err := os.CreateTemp(limits.TempDir, "safedrop-validate-*")
	if err != nil {
		return nil, err
	}
	file := &File{file: temp}
	ok := false
	defer func() {
		if !ok {
			_ = file.Close()
		}
	}()
	file.Size, err = io.Copy(temp, requestReader{r.Context(), r.Body})
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, errors.Join(ErrTooLarge, err)
		}
		return nil, err
	}
	if file.Size < limits.MinBytes {
		return nil, ErrTooSmall
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var header [512]byte
	n, err := io.ReadFull(temp, header[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	file.MIME = detectType(header[:n])
	if file.MIME == "" {
		return nil, ErrType
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	config, format, err := image.DecodeConfig(requestReader{r.Context(), temp})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrImage, err)
	}
	if "image/"+format != file.MIME {
		return nil, ErrType
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, ErrImage
	}
	if uint64(config.Width) > limits.MaxPixels/uint64(config.Height) {
		return nil, ErrPixels
	}
	file.Width, file.Height = config.Width, config.Height
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	ok = true
	return file, nil
}
