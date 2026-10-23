package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/db"
	"safedrop/internal/storage"
)

type variantQueries interface {
	ListFileVariants(context.Context, pgtype.UUID) ([]db.FileVariant, error)
}

type variantView struct {
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
	Size   int64  `json:"size"`
	MIME   string `json:"mime"`
	URL    string `json:"url"`
}

func (h *FilesHandler) view(r *http.Request, f db.File) (fileView, error) {
	view := publicFile(f)
	q, ok := h.Queries.(variantQueries)
	if !ok || f.Status != "ready" {
		return view, nil
	}
	variants, err := q.ListFileVariants(r.Context(), f.ID)
	if err != nil {
		return view, err
	}
	for _, variant := range variants {
		key := "w" + strconv.Itoa(int(variant.TargetWidth))
		if _, ok := view.Variants[key]; ok {
			url := "/v1/files/" + f.ID.String() + "/variants/" + strconv.Itoa(int(variant.TargetWidth))
			view.Variants[key] = &variantView{variant.Width, variant.Height, variant.Size, variant.Mime, url}
		}
	}
	return view, nil
}

func (h *FilesHandler) Variant(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.owner(w, r)
	if !ok {
		return
	}
	var id pgtype.UUID
	if id.Scan(chi.URLParam(r, "id")) != nil || !id.Valid {
		Error(w, "invalid file id", 400)
		return
	}
	f, err := h.Queries.GetOwnerFile(r.Context(), db.GetOwnerFileParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		Error(w, "not found", 404)
		return
	}
	if err != nil {
		Error(w, "file operation failed", 500)
		return
	}
	if f.Status != "ready" {
		Error(w, "file is not ready", 409)
		return
	}
	q, ok := h.Queries.(variantQueries)
	if !ok || h.Storage == nil {
		Error(w, "storage unavailable", 503)
		return
	}
	variants, err := q.ListFileVariants(r.Context(), id)
	if err != nil {
		Error(w, "file operation failed", 500)
		return
	}
	for _, v := range variants {
		if strconv.Itoa(int(v.TargetWidth)) != chi.URLParam(r, "width") {
			continue
		}
		body, err := h.Storage.Get(r.Context(), v.StorageKey)
		if errors.Is(err, storage.ErrNotFound) {
			Error(w, "not found", 404)
			return
		}
		if err != nil {
			Error(w, "storage unavailable", 503)
			return
		}
		defer body.Close()
		w.Header().Set("Content-Type", v.Mime)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Length", strconv.FormatInt(v.Size, 10))
		_, _ = io.Copy(w, body)
		return
	}
	Error(w, "not found", 404)
}
