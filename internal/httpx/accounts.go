package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/auth"
	"safedrop/internal/ratelimit"
)

func WithAccounts(accounts *auth.Accounts, checks ...auth.RegistrationCheck) RouterOption {
	var check auth.RegistrationCheck
	if len(checks) > 0 {
		check = checks[0]
	}
	return func(r chi.Router) {
		r.Post("/v1/auth/register", accountEndpoint(accounts, true, check))
		r.Post("/v1/auth/login", accountEndpoint(accounts, false, nil))
	}
}

func accountEndpoint(accounts *auth.Accounts, register bool, check auth.RegistrationCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		defer r.Body.Close()
		var input struct {
			Email          string `json:"email"`
			Password       string `json:"password"`
			TurnstileToken string `json:"turnstile_token"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			Error(w, "invalid credentials payload", 400)
			return
		}
		if check != nil {
			if err := check.Verify(r.Context(), input.TurnstileToken, ratelimit.IP(r)); err != nil {
				if errors.Is(err, auth.ErrChallenge) {
					Error(w, "registration verification failed", 403)
				} else {
					w.Header().Set("Retry-After", "5")
					Error(w, "registration verification unavailable", 503)
				}
				return
			}
		}
		token, err := accounts.Exchange(r.Context(), input.Email, input.Password, register)
		if err != nil {
			status := 500
			message := "authentication failed"
			switch {
			case errors.Is(err, auth.ErrInput):
				status, message = 400, "invalid email or password length"
			case errors.Is(err, auth.ErrCredentials):
				status, message = 401, "invalid credentials"
			case errors.Is(err, auth.ErrExists):
				status, message = 409, "account already exists"
			case errors.Is(err, auth.ErrBusy):
				status, message = 429, "authentication busy"
				w.Header().Set("Retry-After", "1")
			}
			Error(w, message, status)
			return
		}
		status := http.StatusOK
		if register {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": token})
	}
}
