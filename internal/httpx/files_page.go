package httpx

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
)

type fileView struct {
	ID        pgtype.UUID        `json:"id"`
	Name      string             `json:"original_name"`
	Status    string             `json:"status"`
	MIME      string             `json:"mime"`
	Size      int64              `json:"size"`
	Width     pgtype.Int4        `json:"width"`
	Height    pgtype.Int4        `json:"height"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func publicFile(f db.File) fileView {
	return fileView{f.ID, f.OriginalName, f.Status, f.Mime, f.Size, f.Width, f.Height, f.CreatedAt}
}

func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.owner(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			http.Error(w, "invalid limit", 400)
			return
		}
	}
	p := db.PageOwnerFilesParams{OwnerID: owner, PageSize: int32(limit + 1)}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		if len(raw) > 256 {
			http.Error(w, "invalid cursor", 400)
			return
		}
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		parts := strings.Split(string(decoded), "|")
		if err != nil || len(parts) != 2 {
			http.Error(w, "invalid cursor", 400)
			return
		}
		timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil || p.CursorID.Scan(parts[1]) != nil || !p.CursorID.Valid {
			http.Error(w, "invalid cursor", 400)
			return
		}
		p.HasCursor, p.CursorTime = true, pgtype.Timestamptz{Time: timestamp, Valid: true}
	}
	files, err := h.Queries.PageOwnerFiles(r.Context(), p)
	if err != nil {
		http.Error(w, "file operation failed", 500)
		return
	}
	next := ""
	if len(files) > limit {
		last := files[limit-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.Time.Format(time.RFC3339Nano) + "|" + last.ID.String()))
		files = files[:limit]
	}
	items := make([]fileView, 0, len(files))
	for _, f := range files {
		items = append(items, publicFile(f))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Files []fileView `json:"files"`
		Next  string     `json:"next_cursor,omitempty"`
	}{items, next})
}
