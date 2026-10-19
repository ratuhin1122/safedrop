package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/auth"
)

func WithAccounts(accounts *auth.Accounts) RouterOption {
	return func(r chi.Router) {
		r.Post("/v1/auth/register", accountEndpoint(accounts, true))
		r.Post("/v1/auth/login", accountEndpoint(accounts, false))
	}
}

func accountEndpoint(accounts *auth.Accounts, register bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		defer r.Body.Close()
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			http.Error(w, "invalid credentials payload", 400)
			return
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
			}
			http.Error(w, message, status)
			return
		}
		status := http.StatusOK
		if register {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": 900})
	}
}
