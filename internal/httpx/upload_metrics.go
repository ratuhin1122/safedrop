package httpx

import (
	"errors"
	"safedrop/internal/observability"
	"safedrop/internal/validate"
)

func recordUploadRejection(err error) {
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
		observability.DefaultMetrics.Uploads.WithLabelValues(result).Inc()
	}
}
