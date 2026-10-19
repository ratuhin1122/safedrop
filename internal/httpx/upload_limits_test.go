package httpx

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/ratelimit"
)

func TestTwentyFirstUploadReturns429(t *testing.T) {
	h, _ := testUploadHandler(t)
	h.Admission = ratelimit.NewUploads(ratelimit.UploadConfig{Window: time.Minute, PerUser: 20, PerIP: 20})
	created := 0
	h.Repository = uploadRepositoryFunc(func(_ context.Context, p db.CreateFileParams) (db.File, error) {
		created++
		return db.File{ID: p.OwnerID}, nil
	})
	data := uploadPNG(t)
	for i := 1; i <= 21; i++ {
		w := httptest.NewRecorder()
		r := uploadRequest(data)
		body := &unreadBody{}
		if i == 21 {
			r.Body = body
		}
		h.ServeHTTP(w, r)
		if body.reads != 0 {
			t.Fatal("limited request body was read")
		}
		want := 201
		if i == 21 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("upload %d: got %d, body %s", i, w.Code, w.Body)
		}
		if i == 21 && w.Header().Get("Retry-After") == "" {
			t.Fatal("missing retry header")
		}
	}
	if created != 20 {
		t.Fatalf("stored %d files", created)
	}
}

func TestGeneralRateLimit(t *testing.T) {
	h, _ := testUploadHandler(t)
	router := NewRouter(h.Logger, nil, WithRateLimits(ratelimit.NewBuckets(1, 1)))
	for i, want := range []int{404, 429} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/missing", nil))
		if w.Code != want {
			t.Fatalf("request %d = %d", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
