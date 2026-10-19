package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"safedrop/internal/auth"
	"safedrop/internal/share"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ShareHandler struct {
	Auth    auth.Authenticator
	Service *share.Service
	Logger  *slog.Logger
}

func WithShares(handler *ShareHandler) RouterOption {
	return func(r chi.Router) {
		r.Post("/v1/files/{id}/shares", handler.Create)
		r.Get("/s/{token}", handler.Download)
	}
}

func (h *ShareHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	owner, err := h.Auth.Authenticate(r)
	if err != nil || !owner.Valid || owner.Bytes == [16]byte{} {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var id pgtype.UUID
	if err := id.Scan(chi.URLParam(r, "id")); err != nil || !id.Valid {
		http.Error(w, "invalid file id", 400)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()
	var input struct {
		ExpiresAt    time.Time `json:"expires_at"`
		Password     string    `json:"password"`
		MaxDownloads *int64    `json:"max_downloads"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid share settings", 400)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid trailing data", 400)
		return
	}
	created, err := h.Service.Create(r.Context(), owner, id, share.CreateOptions{
		ExpiresAt: input.ExpiresAt, Password: input.Password, MaxDownloads: input.MaxDownloads})
	if err != nil {
		h.shareError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *ShareHandler) shareError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, share.ErrGone):
		http.Error(w, "link expired or unavailable", 410)
	case errors.Is(err, share.ErrPassword):
		w.Header().Set("WWW-Authenticate", `Basic realm="share", charset="UTF-8"`)
		http.Error(w, "password required or incorrect", 401)
	case errors.Is(err, share.ErrInvalid):
		http.Error(w, "invalid share settings", 400)
	case errors.Is(err, share.ErrNotFound):
		http.Error(w, "file not found", 404)
	default:
		// Never log token URLs or password-bearing requests.
		h.Logger.ErrorContext(r.Context(), "share operation failed")
		http.Error(w, "share operation failed", 500)
	}
}
