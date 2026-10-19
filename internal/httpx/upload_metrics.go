package httpx

import (
	"context"
	"errors"
	"safedrop/internal/observability"
	"safedrop/internal/validate"
)

func (h *UploadHandler) recordUploadRejection(ctx context.Context, err error) {
	result := ""
	switch {
	case errors.Is(err, validate.ErrTooLarge), errors.Is(err, validate.ErrTooSmall):
		result = "rejected_size"
	case errors.Is(err, validate.ErrPixels):
		result = "rejected_pixels"
	case errors.Is(err, validate.ErrType), errors.Is(err, validate.ErrImage):
		result = "rejected_type"
	}
	if result != "" {
		h.Metrics.ObserveUpload(result, 0)
		observability.LogRejection(ctx, h.Logger, "", result)
	}
}
