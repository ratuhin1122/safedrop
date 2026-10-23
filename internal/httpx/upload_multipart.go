package httpx

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"

	"safedrop/internal/validate"
)

// Multipart headers are bounded separately from actual file bytes. No ParseMultipartForm.
func (h *UploadHandler) validateBody(w http.ResponseWriter, r *http.Request) (*validate.File, error) {
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media != "multipart/form-data" {
		return validate.Validate(w, r, h.Limits)
	}
	body := http.MaxBytesReader(w, r.Body, multipartLimit(h.Limits.MaxBytes))
	defer body.Close()
	r.Body = body
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, validate.ErrImage
	}
	part, err := reader.NextPart()
	if err != nil {
		return nil, multipartError(err)
	}
	// Header.Filename is parsed directly: Part.FileName strips unsafe path prefixes.
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil || part.FormName() != "file" {
		return nil, validate.ErrImage
	}
	r.Header.Set("X-Filename", params["filename"])
	r.ContentLength = -1
	r.Body = part
	if _, _, err := checkRawMultipartName(r, h.Limits); err != nil {
		return nil, errors.Join(validate.ErrImage, err)
	}
	file, err := validate.Validate(w, r, h.Limits)
	if err != nil {
		return nil, multipartError(err)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		_ = file.Close()
		if err != nil {
			return nil, multipartError(err)
		}
		return nil, validate.ErrImage
	}
	return file, nil
}

func checkRawMultipartName(r *http.Request, limits validate.Limits) (string, int, error) {
	copy := r.Clone(r.Context())
	copy.Header = r.Header.Clone()
	copy.Header.Del("Content-Type")
	return checkUpload(copy, limits)
}

func multipartLimit(max int64) int64 {
	if max > (1<<63-1)-65536 {
		return 1<<63 - 1
	}
	return max + 65536
}

func multipartError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return errors.Join(validate.ErrTooLarge, err)
	}
	if errors.Is(err, multipart.ErrMessageTooLarge) {
		return validate.ErrImage
	}
	return errors.Join(validate.ErrImage, err)
}
