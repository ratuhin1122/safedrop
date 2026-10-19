package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/observability"
)

type FileQueries interface {
	GetOwnerFile(context.Context, db.GetOwnerFileParams) (db.File, error)
	PageOwnerFiles(context.Context, db.PageOwnerFilesParams) ([]db.File, error)
	SoftDeleteFile(context.Context, db.SoftDeleteFileParams) (int64, error)
}

type FilesHandler struct {
	Auth    auth.Authenticator
	Queries FileQueries
}

func WithFiles(h *FilesHandler) RouterOption {
	return func(r chi.Router) {
		r.Get("/v1/files", h.List)
		r.Get("/v1/files/{id}", h.One)
		r.Delete("/v1/files/{id}", h.One)
	}
}

func (h *FilesHandler) owner(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	w.Header().Set("Cache-Control", "no-store")
	id, err := h.Auth.Authenticate(r)
	if err != nil || !id.Valid || id.Bytes == [16]byte{} {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "authentication required", 401)
		return id, false
	}
	observability.SetUserID(r.Context(), id.String())
	return id, true
}

func (h *FilesHandler) One(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.owner(w, r)
	if !ok {
		return
	}
	var id pgtype.UUID
	if id.Scan(chi.URLParam(r, "id")) != nil || !id.Valid {
		http.Error(w, "invalid file id", 400)
		return
	}
	if r.Method == http.MethodDelete {
		n, err := h.Queries.SoftDeleteFile(r.Context(), db.SoftDeleteFileParams{ID: id, OwnerID: owner})
		if err != nil {
			http.Error(w, "file operation failed", 500)
			return
		}
		if n == 0 {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	f, err := h.Queries.GetOwnerFile(r.Context(), db.GetOwnerFileParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "file operation failed", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(publicFile(f))
}
