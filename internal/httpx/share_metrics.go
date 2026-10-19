package httpx

import (
	"errors"
	"safedrop/internal/observability"
	"safedrop/internal/share"
)

func recordShareRequest(err error) {
	result := ""
	switch {
	case err == nil:
		result = "ok"
	case errors.Is(err, share.ErrExpired):
		result = "expired"
	case errors.Is(err, share.ErrLimitReached):
		result = "limit_reached"
	case errors.Is(err, share.ErrPassword):
		result = "wrong_password"
	case errors.Is(err, share.ErrGone), errors.Is(err, share.ErrNotFound):
		result = "not_found"
	}
	if result != "" {
		observability.DefaultMetrics.Shares.WithLabelValues(result).Inc()
	}
}
