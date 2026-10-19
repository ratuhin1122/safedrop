package httpx

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	"safedrop/internal/db"
	"safedrop/internal/ratelimit"
)

func TestUploadQuotaIsAtomic(t *testing.T) {
	h, _, pool, user := integrationUpload(t)
	data := uploadPNG(t)
	repository := h.Repository.(*UploadDatabase)
	repository.DailyBytes = int64(len(data))
	// No advisory precheck: concurrent commits must enforce the quota themselves.
	const attempts = 8
	codes := make(chan int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := uploadRequest(data)
			r.ContentLength = -1
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code == 429 && w.Header().Get("Retry-After") == "" {
				t.Error("missing retry header")
			}
			codes <- w.Code
		}()
	}
	wg.Wait()
	close(codes)
	ok, denied := 0, 0
	for code := range codes {
		switch code {
		case 201:
			ok++
		case 429:
			denied++
		default:
			t.Errorf("status=%d", code)
		}
	}
	if ok != 1 || denied != attempts-1 {
		t.Fatalf("created=%d limited=%d", ok, denied)
	}
	ctx := context.Background()
	var bytes, count, files, jobs int64
	if err := pool.QueryRow(ctx, "SELECT bytes,count FROM upload_usage WHERE user_id=$1", user.ID).Scan(&bytes, &count); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM files WHERE owner_id=$1", user.ID).Scan(&files); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE args->>'file_id' IN (SELECT id::text FROM files WHERE owner_id=$1)", user.ID).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if bytes != int64(len(data)) || count != 1 || files != 1 || jobs != 1 {
		t.Fatalf("bytes=%d count=%d files=%d jobs=%d", bytes, count, files, jobs)
	}
	h.Quota = &ratelimit.Quota{Queries: db.New(pool), Bytes: repository.DailyBytes}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, uploadRequest(data))
	if w.Code != 429 {
		t.Fatal(w.Code)
	}
}
