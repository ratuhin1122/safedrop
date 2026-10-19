package httpx

import (
	"net/http"
	"safedrop/internal/observability"
	"safedrop/internal/validate"
)

func (h *UploadHandler) validateUpload(w http.ResponseWriter, r *http.Request) (*validate.File, error) {
	ctx, end := observability.Span(r.Context(), "validate.upload")
	file, err := validate.Validate(w, r.WithContext(ctx), h.Limits)
	end(err)
	return file, err
}
