package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testS3(t *testing.T, endpoint string) *S3 {
	t.Helper()
	s, err := NewS3(S3Config{Endpoint: endpoint, Region: "us-east-1", Bucket: "private",
		AccessKeyID: "test-key", SecretAccessKey: "test-secret", ForcePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestS3StreamingMultipart(t *testing.T) {
	var parts, bytes atomic.Int64
	var completed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/private/incoming/large" {
			t.Errorf("path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			_, _ = io.WriteString(w, "<InitiateMultipartUploadResult><UploadId>test</UploadId></InitiateMultipartUploadResult>")
		case r.Method == http.MethodPut && q.Get("uploadId") == "test":
			n, err := io.Copy(io.Discard, r.Body)
			if err != nil {
				t.Error(err)
			}
			bytes.Add(n)
			parts.Add(1)
			w.Header().Set("ETag", `"part"`)
		case r.Method == http.MethodPost && q.Get("uploadId") == "test":
			completed.Store(true)
			_, _ = io.WriteString(w, "<CompleteMultipartUploadResult><ETag>done</ETag></CompleteMultipartUploadResult>")
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	const size = 11 << 20
	if err := testS3(t, server.URL).Put(context.Background(), "incoming/large", io.LimitReader(&zeroReader{}, size), "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	if bytes.Load() != size || parts.Load() != 3 || !completed.Load() {
		t.Fatalf("multipart bytes=%d parts=%d complete=%v", bytes.Load(), parts.Load(), completed.Load())
	}
}

func TestS3Presign(t *testing.T) {
	s := testS3(t, "https://storage.example.test")
	signed, err := s.PresignGet(context.Background(), "final/file", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "storage.example.test" || u.Path != "/private/final/file" || u.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("unexpected signed URL: %s", signed)
	}
	if _, err := s.PresignGet(context.Background(), "incoming/file", time.Minute); err == nil {
		t.Fatal("presigned incoming object")
	}
}

func TestS3AbortsFailedMultipart(t *testing.T) {
	var aborted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			_, _ = io.WriteString(w, "<InitiateMultipartUploadResult><UploadId>test</UploadId></InitiateMultipartUploadResult>")
		case http.MethodDelete:
			aborted.Store(true)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "<Error><Code>InvalidRequest</Code></Error>")
		}
	}))
	defer server.Close()
	err := testS3(t, server.URL).Put(context.Background(), "incoming/fail", io.LimitReader(&zeroReader{}, 6<<20), "")
	if err == nil || !aborted.Load() {
		t.Fatalf("error=%v abort=%v", err, aborted.Load())
	}
}

func TestS3Configuration(t *testing.T) {
	if _, err := NewS3(S3Config{}); err == nil {
		t.Fatal("accepted empty config")
	}
	if _, err := NewS3(S3Config{Bucket: "b", Region: "r", AccessKeyID: "a", SecretAccessKey: "s", Endpoint: "file:///tmp"}); err == nil {
		t.Fatal("accepted non-HTTP endpoint")
	}
	s := testS3(t, "https://storage.example.test")
	if err := s.Put(context.Background(), "../escape", strings.NewReader("x"), ""); err == nil {
		t.Fatal("accepted unsafe key")
	}
}
