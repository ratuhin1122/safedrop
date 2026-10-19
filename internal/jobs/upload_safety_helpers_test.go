package jobs

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"safedrop/internal/db"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

// A deterministic protocol fixture, not a replacement for live ClamAV tests.
func signatureClamd(t *testing.T, signature []byte) *scan.Client {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				var command [10]byte
				if _, err := io.ReadFull(conn, command[:]); err != nil || string(command[:]) != "zINSTREAM\x00" {
					t.Error("invalid clamd command")
					return
				}
				detector := &signatureWriter{signature: signature}
				for {
					var header [4]byte
					if _, err := io.ReadFull(conn, header[:]); err != nil {
						t.Error(err)
						return
					}
					size := binary.BigEndian.Uint32(header[:])
					if size == 0 {
						break
					}
					if size > 4096 {
						t.Error("oversized clamd chunk")
						return
					}
					if _, err := io.CopyN(detector, conn, int64(size)); err != nil {
						t.Error(err)
						return
					}
				}
				reply := "stream: OK\x00"
				if detector.found {
					reply = "stream: Eicar-Test FOUND\x00"
				}
				if _, err := io.WriteString(conn, reply); err != nil {
					t.Error(err)
				}
			}()
		}
	}()
	t.Cleanup(func() { _ = listener.Close(); wg.Wait() })
	client, err := scan.NewClient(listener.Addr().String(), 5*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertScannedUpload(t *testing.T, ctx context.Context, pool *pgxpool.Pool, store storage.Store, original db.File, infected bool) {
	t.Helper()
	file, err := db.New(pool).GetFile(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	want, variants := "ready", 3
	if infected {
		want, variants = "rejected", 0
	}
	if file.Status != want {
		t.Fatalf("status=%s want=%s", file.Status, want)
	}
	body, err := store.Get(ctx, original.StorageKey)
	if body != nil {
		body.Close()
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("incoming object was not deleted: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM file_variants WHERE file_id=$1", file.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != variants {
		t.Fatalf("variants=%d want=%d", count, variants)
	}
	if infected {
		for _, width := range []string{"375", "768", "1200"} {
			body, err := store.Get(ctx, "final/"+file.ID.String()+"/"+width+".webp")
			if body != nil {
				body.Close()
			}
			if !errors.Is(err, storage.ErrNotFound) {
				t.Fatalf("infected file published: %v", err)
			}
		}
	}
}
