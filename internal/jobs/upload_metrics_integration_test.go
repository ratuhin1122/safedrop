package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"safedrop/internal/auth"
	"safedrop/internal/db"
	"safedrop/internal/httpx"
	"safedrop/internal/imageproc"
	"safedrop/internal/observability"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
	"safedrop/internal/validate"
)

// The test scanner detects the complete EICAR string with bounded memory.
type signatureWriter struct {
	signature []byte
	matched   int
	found     bool
}

func (w *signatureWriter) Write(data []byte) (int, error) {
	for _, b := range data {
		if b == w.signature[w.matched] {
			w.matched++
		} else {
			w.matched = 0
			if b == w.signature[0] {
				w.matched = 1
			}
		}
		if w.matched == len(w.signature) {
			w.found = true
			w.matched = 0
		}
	}
	return len(data), nil
}

func TestUploadAndScanShareInjectedMetrics(t *testing.T) {
	ctx, pool, fixture, logger, _ := workerFixture(t)
	m := observability.NewMetrics()
	store, err := storage.NewLocal(storage.LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	queue, err := NewQueue(pool, logger)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("t", 32)
	authenticator, err := auth.NewStub(fixture.OwnerID.String(), token)
	if err != nil {
		t.Fatal(err)
	}
	upload := httpx.NewUploadHandler(httpx.UploadHandler{Auth: authenticator, Storage: store, Logger: logger,
		Repository: &httpx.UploadDatabase{Pool: pool, Queue: queue}, Limits: validate.Limits{MinBytes: 1, MaxBytes: 4096, MaxPixels: 100}}, m)
	engine, err := imageproc.NewWithEncoder(imageproc.Options{MaxBytes: 4096, MaxPixels: 20000000}, &imageTestEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	eicar, png := scanFixtures(t)
	pipeline := &ScanProcessor{Storage: store, Logger: logger, Next: &ImageProcessor{Engine: engine, Storage: store, Pool: pool},
		Scanner: scannerFunc(func(_ context.Context, body io.Reader) (scan.Result, error) {
			detector := &signatureWriter{signature: eicar}
			if _, err := io.Copy(detector, body); err != nil {
				return scan.Result{}, err
			}
			if detector.found {
				return scan.Result{Verdict: scan.Infected, Signature: "Eicar-Test"}, nil
			}
			return scan.Result{Verdict: scan.Clean}, nil
		})}
	worker := NewProcessFile(pool, pipeline, logger, 0, m)
	// Appending EICAR keeps PNG validation valid while exercising background rejection.
	infected := append(append([]byte{}, png...), eicar...)
	for _, payload := range [][]byte{png, infected} {
		r := httptest.NewRequest("POST", "/v1/files", bytes.NewReader(payload))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Filename", "image.png")
		w := httptest.NewRecorder()
		upload.ServeHTTP(w, r)
		if w.Code != 201 {
			t.Fatalf("upload: %d %s", w.Code, w.Body)
		}
		var result struct {
			ID pgtype.UUID `json:"id"`
		}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		file, err := db.New(pool).GetFile(ctx, result.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := worker.Work(ctx, testJob(file, 1, 5)); err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	m.Handler(QueueMetrics(pool)).ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	for _, line := range []string{`safedrop_uploads_total{result="accepted"} 2`,
		`safedrop_files_processed_total{result="ready"} 1`, `safedrop_files_processed_total{result="rejected_virus"} 1`,
		`safedrop_processing_duration_seconds_count{stage="scan"} 2`,
		`safedrop_processing_duration_seconds_count{stage="convert"} 1`,
		`safedrop_processing_duration_seconds_count{stage="total"} 2`} {
		if !strings.Contains(w.Body.String(), line) {
			t.Errorf("missing %s", line)
		}
	}
}
