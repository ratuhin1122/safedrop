package httpx

import (
	"context"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"safedrop/internal/db"
	"safedrop/internal/ratelimit"
)

func assertUploadCounts(t *testing.T, pool *pgxpool.Pool, user db.User, want int) {
	t.Helper()
	var files, jobs, usage int
	err := pool.QueryRow(context.Background(), `SELECT
	 (SELECT count(*) FROM files WHERE owner_id=$1),
	 (SELECT count(*) FROM river_job WHERE args->>'file_id' IN (SELECT id::text FROM files WHERE owner_id=$1)),
	 (SELECT COALESCE(sum(count),0) FROM upload_usage WHERE user_id=$1)`, user.ID).Scan(&files, &jobs, &usage)
	if err != nil {
		t.Fatal(err)
	}
	if files != want || jobs != want || usage != want {
		t.Fatalf("files=%d jobs=%d usage=%d want=%d", files, jobs, usage, want)
	}
}

func TestUploadWindowPostgres(t *testing.T) {
	h, _, pool, user := integrationUpload(t)
	h.Admission = ratelimit.NewUploads(ratelimit.UploadConfig{Window: time.Hour, PerUser: 20, PerIP: 20})
	router := NewRouter(h.Logger, nil, nil, WithUploads(h))
	for i := 1; i <= 21; i++ {
		r := uploadRequest(uploadPNG(t))
		unread := &unreadBody{}
		if i == 21 {
			r.Body = unread
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		want := 201
		if i == 21 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("upload %d: status=%d body=%s", i, w.Code, w.Body)
		}
		if i == 21 {
			retry, err := strconv.Atoi(w.Header().Get("Retry-After"))
			if err != nil || retry < 1 || unread.reads != 0 {
				t.Fatal("invalid retry header or denied body was read")
			}
		}
	}
	assertUploadCounts(t, pool, user, 20)
}
