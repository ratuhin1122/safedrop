package scan

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func eicar() string {
	return strings.Join([]string{"X5O!P%@AP[4", "\\PZX54(P^)7CC)7}$", "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"}, "")
}

func normalPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func mockClamd(t *testing.T, handler func(net.Conn)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			group.Add(1)
			go func() {
				defer group.Done()
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				handler(conn)
			}()
		}
	}()
	t.Cleanup(func() { _ = listener.Close(); group.Wait() })
	return listener.Addr().String()
}

func readStream(conn net.Conn) (int64, string, error) {
	var command [10]byte
	if _, err := io.ReadFull(conn, command[:]); err != nil {
		return 0, "", err
	}
	if string(command[:]) != "zINSTREAM\x00" {
		return 0, "", fmt.Errorf("wrong command: %q", command)
	}
	hash := md5.New() // Test signature comparison only, not a security primitive.
	var total int64
	for {
		var header [4]byte
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			return 0, "", err
		}
		size := binary.BigEndian.Uint32(header[:])
		if size == 0 {
			return total, hex.EncodeToString(hash.Sum(nil)), nil
		}
		if size > 32<<10 {
			return 0, "", fmt.Errorf("oversized chunk: %d", size)
		}
		n, err := io.CopyN(hash, conn, int64(size))
		if err != nil {
			return 0, "", err
		}
		total += n
	}
}
