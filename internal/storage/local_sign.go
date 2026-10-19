package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (l *Local) signature(key, expires string) []byte {
	mac := hmac.New(sha256.New, l.signingKey)
	_, _ = mac.Write([]byte(key + "\n" + expires))
	return mac.Sum(nil)
}

func (l *Local) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := validateDownload(key, ttl); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if l.baseURL == nil {
		return "", ErrPresignDisabled
	}
	expires := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	u := *l.baseURL
	q := u.Query()
	q.Set("key", key)
	q.Set("expires", expires)
	q.Set("signature", hex.EncodeToString(l.signature(key, expires)))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ServeHTTP serves signed final/ objects, never a directory listing or incoming/.
// Mount it at exactly LocalConfig.BaseURL's path. No other file server is needed.
func (l *Local) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.baseURL == nil || r.URL.Path != l.baseURL.Path {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	key, expires := q.Get("key"), q.Get("expires")
	deadline, err := strconv.ParseInt(expires, 10, 64)
	signature, signatureErr := hex.DecodeString(q.Get("signature"))
	if err != nil || signatureErr != nil || time.Now().Unix() >= deadline ||
		validateKey(key) != nil || !strings.HasPrefix(key, "final/") ||
		!hmac.Equal(signature, l.signature(key, expires)) {
		http.Error(w, "invalid or expired download", http.StatusForbidden)
		return
	}
	body, err := l.Get(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer body.Close()
	file := body.(*localReader).file
	info, err := file.Stat()
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, "", info.ModTime(), file)
}
