package share_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"safedrop/internal/share"
)

func TestPostgresGoneResponses(t *testing.T) {
	for _, expired := range []bool{false, true} {
		ctx, pool, store, file, _ := databaseFixture(t)
		service := &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store}
		max := int64(1)
		token := newLink(t, ctx, service, file, &max)
		if expired {
			expireLink(t, ctx, pool, token)
		} else {
			download, err := service.Open(ctx, token, "")
			if err != nil {
				t.Fatal(err)
			}
			download.Body.Close()
		}
		w := httptest.NewRecorder()
		router(service, io.Discard).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/s/"+token, nil))
		if w.Code != 410 {
			t.Fatalf("expired=%v status=%d", expired, w.Code)
		}
		var downloads int
		if err := pool.QueryRow(ctx, "SELECT download_count FROM share_links WHERE file_id=$1", file.ID).Scan(&downloads); err != nil {
			t.Fatal(err)
		}
		want := 1
		if expired {
			want = 0
		}
		if downloads != want {
			t.Fatalf("denied link changed download count: %d want %d", downloads, want)
		}
	}
}

func TestPostgresConcurrentLimit(t *testing.T) {
	ctx, pool, store, file, _ := databaseFixture(t)
	service := &share.Service{Repository: &share.Postgres{Pool: pool}, Storage: store}
	max := int64(1)
	token := newLink(t, ctx, service, file, &max)
	handler := router(service, io.Discard)
	var success, gone atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/s/"+token, nil))
			if w.Code == 200 {
				success.Add(1)
			} else if w.Code == 410 {
				gone.Add(1)
			}
		}()
	}
	group.Wait()
	if success.Load() != 1 || gone.Load() != 7 {
		t.Fatalf("success=%d gone=%d", success.Load(), gone.Load())
	}
}
