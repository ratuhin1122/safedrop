package httpx

import (
	"errors"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"safedrop/internal/validate"
)

// These checks run after authentication and before the first body read.
// X-Filename is display metadata; it never determines type or storage paths.
func checkUpload(r *http.Request, limits validate.Limits) (string, int, error) {
	values := r.Header.Values("X-Filename")
	if len(values) != 1 {
		return "", http.StatusBadRequest, errors.New("one X-Filename header is required")
	}
	name := strings.TrimSpace(values[0])
	if name == "" || name == "." || name == ".." || len(name) > 255 || !utf8.ValidString(name) ||
		strings.ContainsAny(name, "/\\") || strings.ContainsFunc(name, unicode.IsControl) {
		return "", http.StatusBadRequest, errors.New("invalid X-Filename")
	}
	if r.ContentLength > limits.MaxBytes {
		return "", http.StatusRequestEntityTooLarge, errors.New("upload exceeds the maximum size")
	}
	if r.Body == nil || r.ContentLength < -1 || (r.ContentLength >= 0 && r.ContentLength < limits.MinBytes) {
		return "", http.StatusBadRequest, errors.New("upload is smaller than the minimum size")
	}
	return name, 0, nil
}
