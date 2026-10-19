package share_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"safedrop/internal/share"
)

func TestCreateShareHTTP(t *testing.T) {
	service, repo, _, _ := fixture(t)
	data, _ := json.Marshal(map[string]any{"expires_at": time.Now().Add(time.Hour), "max_downloads": 2})
	r := httptest.NewRequest(http.MethodPost, "/v1/files/"+repo.file.ID.String()+"/shares", bytes.NewReader(data))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	w := httptest.NewRecorder()
	router(service, io.Discard).ServeHTTP(w, r)
	var created share.Created
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || !strings.HasPrefix(created.URL, "/s/") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
