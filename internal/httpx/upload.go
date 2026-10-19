package httpx

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/ratelimit"
	"safedrop/internal/storage"
	"safedrop/internal/validate"
)

type UploadHandler struct {
	Auth       auth.Authenticator
	Storage    storage.Store
	Repository UploadRepository
	Limits     validate.Limits
	Logger     *slog.Logger
	Admission  ratelimit.UploadLimiter
	Quota      *ratelimit.Quota
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner, err := h.Auth.Authenticate(r)
	if err != nil || !owner.Valid || owner.Bytes == [16]byte{} {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	name, status, err := checkUpload(r, h.Limits)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	release, ok := h.admit(w, r, owner)
	if !ok {
		return
	}
	defer release()
	file, err := validate.Validate(w, r, h.Limits)
	if err != nil {
		h.uploadError(w, r, err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.Logger.ErrorContext(r.Context(), "upload temp cleanup failed", "error", err)
		}
	}()
	if h.Quota != nil {
		if err := h.Quota.Check(r.Context(), owner, file.Size); err != nil {
			h.uploadError(w, r, err)
			return
		}
	}
	key := "incoming/" + rand.Text()
	keep := false
	defer func() {
		if !keep {
			h.cleanupUpload(r.Context(), key)
		}
	}()
	if err := h.Storage.Put(r.Context(), key, file, file.MIME); err != nil {
		h.uploadError(w, r, err)
		return
	}
	row, err := h.Repository.CreateUpload(r.Context(), db.CreateFileParams{
		OwnerID: owner, OriginalName: name, Mime: file.MIME, Size: file.Size, StorageKey: key,
	})
	if err != nil {
		keep = errors.Is(err, ErrUploadCommitUncertain)
		h.uploadError(w, r, err)
		return
	}
	keep = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(struct {
		ID string `json:"id"`
	}{ID: row.ID.String()})
}

func (h *UploadHandler) cleanupUpload(ctx context.Context, key string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := h.Storage.Delete(ctx, key); err != nil {
		h.Logger.ErrorContext(ctx, "incoming upload cleanup failed", "key", key, "error", err)
	}
}

func (h *UploadHandler) uploadError(w http.ResponseWriter, r *http.Request, err error) {
	if limited(w, err) {
		return
	}
	status, message := http.StatusInternalServerError, "upload failed"
	switch {
	case errors.Is(err, validate.ErrTooLarge):
		status, message = http.StatusRequestEntityTooLarge, "upload exceeds the maximum size"
	case errors.Is(err, validate.ErrType):
		status, message = http.StatusUnsupportedMediaType, "unsupported image type"
	case errors.Is(err, validate.ErrTooSmall), errors.Is(err, validate.ErrImage), errors.Is(err, validate.ErrPixels):
		status, message = http.StatusBadRequest, "invalid image or dimensions"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusRequestTimeout, "upload canceled or timed out"
	}
	if status == http.StatusInternalServerError {
		h.Logger.ErrorContext(r.Context(), "upload failed", "error", err)
	}
	http.Error(w, message, status)
}
