package httpx

import (
	"net/http"
	"safedrop/internal/observability"
	"safedrop/internal/validate"
)

func (h *UploadHandler) validateUpload(w http.ResponseWriter, r *http.Request) (*validate.File, error) {
	ctx, end := observability.Span(r.Context(), "validate.upload")
	file, err := h.validateBody(w, r.WithContext(ctx))
	end(err)
	return file, err
}
