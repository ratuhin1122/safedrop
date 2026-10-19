package httpx

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/ratelimit"
)

func WithRateLimits(buckets *ratelimit.Buckets) RouterOption {
	return func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/healthz" && !(r.Method == http.MethodPost && r.URL.Path == "/v1/files") {
					if retry := buckets.Allow(ratelimit.IP(r)); retry > 0 {
						ratelimit.Respond(w, retry)
						return
					}
				}
				next.ServeHTTP(w, r)
			})
		})
	}
}

func (h *UploadHandler) admit(w http.ResponseWriter, r *http.Request, owner pgtype.UUID) (func(), bool) {
	release := func() {}
	var err error
	if h.Admission != nil {
		release, err = h.Admission.Begin(r.Context(), owner.String(), ratelimit.IP(r))
	}
	if err == nil && h.Quota != nil {
		err = h.Quota.Check(r.Context(), owner, r.ContentLength)
	}
	if err != nil {
		if release != nil {
			release()
		}
		h.uploadError(w, r, err)
		return nil, false
	}
	return release, true
}

func limited(w http.ResponseWriter, err error) bool {
	var rejection *ratelimit.Rejection
	if !errors.As(err, &rejection) {
		return false
	}
	ratelimit.Respond(w, rejection.RetryAfter)
	return true
}
