package validate

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

type heicDimensions struct {
	width, height uint32
	boxes         int
}

// Conservatively bound every declared item extent, including tiles and auxiliary
// images. This is not an HEVC bitstream decoder or a primary-item selector.
func heicConfig(r io.Reader) (image.Config, error) {
	var dimensions heicDimensions
	if err := dimensions.scan(r, "root", 0); err != nil {
		return image.Config{}, err
	}
	if dimensions.width == 0 || dimensions.height == 0 {
		return image.Config{}, image.ErrFormat
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: int(dimensions.width), Height: int(dimensions.height)}, nil
}

func (d *heicDimensions) scan(r io.Reader, parent string, depth int) error {
	if depth > 4 {
		return image.ErrFormat
	}
	for {
		var header [8]byte
		_, err := io.ReadFull(r, header[:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		d.boxes++
		if d.boxes > 4096 {
			return image.ErrFormat
		}
		size, headerSize := uint64(binary.BigEndian.Uint32(header[:4])), uint64(8)
		kind := string(header[4:8])
		if size == 0 && parent == "root" && kind == "mdat" {
			_, err := io.Copy(io.Discard, r)
			return err
		}
		if size == 1 {
			if _, err := io.ReadFull(r, header[:]); err != nil {
				return err
			}
			size, headerSize = binary.BigEndian.Uint64(header[:]), 16
		}
		if size < headerSize || size-headerSize > 1<<63-1 {
			return image.ErrFormat
		}
		body := &io.LimitedReader{R: r, N: int64(size - headerSize)}
		switch {
		case parent == "root" && kind == "meta":
			var fullBox [4]byte
			if _, err := io.ReadFull(body, fullBox[:]); err != nil {
				return err
			}
			if fullBox != [4]byte{} {
				return image.ErrFormat
			}
			if err := d.scan(body, "meta", depth+1); err != nil {
				return err
			}
		case (parent == "meta" && kind == "iprp") || (parent == "iprp" && kind == "ipco"):
			if err := d.scan(body, kind, depth+1); err != nil {
				return err
			}
		case parent == "ipco" && kind == "ispe":
			if err := d.extents(body); err != nil {
				return err
			}
		}
		if _, err := io.Copy(io.Discard, body); err != nil {
			return err
		}
		if body.N != 0 {
			return io.ErrUnexpectedEOF
		}
	}
}

func (d *heicDimensions) extents(r *io.LimitedReader) error {
	if r.N != 12 {
		return image.ErrFormat
	}
	var data [12]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return err
	}
	if binary.BigEndian.Uint32(data[:4]) != 0 {
		return image.ErrFormat
	}
	w, h := binary.BigEndian.Uint32(data[4:8]), binary.BigEndian.Uint32(data[8:12])
	if w == 0 || h == 0 {
		return image.ErrFormat
	}
	// Reject extents that cannot be represented by image.Config on 32-bit Go.
	if uint64(w) > uint64(^uint(0)>>1) || uint64(h) > uint64(^uint(0)>>1) {
		return image.ErrFormat
	}
	if w > d.width {
		d.width = w
	}
	if h > d.height {
		d.height = h
	}
	return nil
}
