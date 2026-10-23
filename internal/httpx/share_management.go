package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
	"safedrop/internal/observability"
)

func (h *ShareHandler) managementOwner(w http.ResponseWriter, r *http.Request) (pgtype.UUID, pgtype.UUID, bool) {
	w.Header().Set("Cache-Control", "no-store")
	owner, err := h.Auth.Authenticate(r)
	if err != nil || !owner.Valid || owner.Bytes == [16]byte{} {
		w.Header().Set("WWW-Authenticate", "Bearer")
		Error(w, "authentication required", 401)
		return owner, pgtype.UUID{}, false
	}
	observability.SetUserID(r.Context(), owner.String())
	var id pgtype.UUID
	if id.Scan(chi.URLParam(r, "id")) != nil || !id.Valid {
		Error(w, "invalid id", 400)
		return owner, id, false
	}
	if h.Queries == nil {
		Error(w, "database unavailable", 503)
		return owner, id, false
	}
	return owner, id, true
}

func (h *ShareHandler) List(w http.ResponseWriter, r *http.Request) {
	owner, id, ok := h.managementOwner(w, r)
	if !ok {
		return
	}
	_, err := h.Queries.GetOwnerFile(r.Context(), db.GetOwnerFileParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		Error(w, "not found", 404)
		return
	}
	if err != nil {
		Error(w, "share operation failed", 500)
		return
	}
	rows, err := h.Queries.ListOwnerShares(r.Context(), db.ListOwnerSharesParams{FileID: id, OwnerID: owner})
	if err != nil {
		Error(w, "share operation failed", 500)
		return
	}
	type item struct {
		ID                pgtype.UUID        `json:"id"`
		FileID            pgtype.UUID        `json:"file_id"`
		ExpiresAt         pgtype.Timestamptz `json:"expires_at"`
		MaxDownloads      pgtype.Int8        `json:"max_downloads"`
		DownloadCount     int64              `json:"download_count"`
		PasswordProtected bool               `json:"password_protected"`
	}
	items := make([]item, 0, len(rows))
	for _, v := range rows {
		items = append(items, item{v.ID, v.FileID, v.ExpiresAt, v.MaxDownloads, v.DownloadCount, v.PasswordProtected})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Items []item `json:"items"`
	}{items})
}

func (h *ShareHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	owner, id, ok := h.managementOwner(w, r)
	if !ok {
		return
	}
	n, err := h.Queries.DeleteOwnerShare(r.Context(), db.DeleteOwnerShareParams{ID: id, OwnerID: owner})
	if err != nil {
		Error(w, "share operation failed", 500)
		return
	}
	if n == 0 {
		Error(w, "not found", 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
