package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"safedrop/internal/db"
)

func WithUsage(files *FilesHandler, queries *db.Queries, bytesLimit int64, uploadsLimit int) RouterOption {
	return func(r chi.Router) {
		r.Get("/v1/me/usage", func(w http.ResponseWriter, r *http.Request) {
			owner, ok := files.owner(w, r)
			if !ok {
				return
			}
			usage, err := queries.OwnerUsage(r.Context(), owner)
			if err != nil {
				Error(w, "usage unavailable", 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(struct {
				BytesToday      int64 `json:"bytes_today"`
				BytesLimit      int64 `json:"bytes_limit"`
				UploadsThisHour int64 `json:"uploads_this_hour"`
				UploadsLimit    int   `json:"uploads_limit"`
			}{usage.BytesToday, bytesLimit, usage.UploadsThisHour, uploadsLimit})
		})
	}
}
