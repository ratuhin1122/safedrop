package httpx

import (
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *ShareHandler) Download(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	_, password, _ := r.BasicAuth()
	download, err := h.Service.Open(r.Context(), chi.URLParam(r, "token"), password)
	h.recordShareRequest(err)
	if err != nil {
		h.shareError(w, r, err)
		return
	}
	if download.URL != "" {
		http.Redirect(w, r, download.URL, http.StatusTemporaryRedirect)
		return
	}
	defer download.Body.Close()
	w.Header().Set("Content-Type", download.MIME)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": download.Name}))
	w.Header().Set("Content-Length", strconv.FormatInt(download.Size, 10))
	if _, err := io.Copy(w, download.Body); err != nil {
		h.Logger.WarnContext(r.Context(), "share stream interrupted")
	}
}
