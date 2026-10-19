package scan

import (
	"context"
	"encoding/binary"
	"io"
)

func sendStream(ctx context.Context, target io.Writer, source io.Reader, maxBytes int64) error {
	var data [32 << 10]byte
	var length [4]byte
	var total int64
	emptyReads := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := source.Read(data[:])
		if n > 0 {
			emptyReads = 0
			if int64(n) > maxBytes-total {
				return ErrTooLarge
			}
			total += int64(n)
			binary.BigEndian.PutUint32(length[:], uint32(n))
			if err := writeAll(target, length[:]); err != nil {
				return err
			}
			if err := writeAll(target, data[:n]); err != nil {
				return err
			}
		} else if err == nil {
			emptyReads++
			if emptyReads >= 100 {
				return io.ErrNoProgress
			}
		}
		if err == io.EOF {
			return writeAll(target, []byte{0, 0, 0, 0})
		}
		if err != nil {
			return err
		}
	}
}

func writeAll(target io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := target.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
