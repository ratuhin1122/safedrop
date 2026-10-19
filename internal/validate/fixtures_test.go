package validate

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func pngFixture(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func jpegFixture(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 3)), nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func hugePNG(t *testing.T) []byte {
	data := pngFixture(t)
	binary.BigEndian.PutUint32(data[16:20], 100000)
	binary.BigEndian.PutUint32(data[20:24], 100000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}

// WebP and HEIC fixtures exercise metadata only, not compressed pixel decoding.
func webpFixture(kind string, width, height uint32) []byte {
	payload := make([]byte, 10)
	switch kind {
	case "VP8X":
		w, h := width-1, height-1
		copy(payload[4:7], []byte{byte(w), byte(w >> 8), byte(w >> 16)})
		copy(payload[7:10], []byte{byte(h), byte(h >> 8), byte(h >> 16)})
	case "VP8L":
		payload = make([]byte, 5)
		payload[0] = 0x2f
		binary.LittleEndian.PutUint32(payload[1:], (width-1)|((height-1)<<14))
	case "VP8 ":
		copy(payload[3:6], []byte{0x9d, 0x01, 0x2a})
		binary.LittleEndian.PutUint16(payload[6:8], uint16(width))
		binary.LittleEndian.PutUint16(payload[8:10], uint16(height))
	}
	data := make([]byte, 20)
	copy(data, "RIFF")
	copy(data[8:], "WEBP"+kind)
	binary.LittleEndian.PutUint32(data[16:], uint32(len(payload)))
	data = append(data, payload...)
	if len(payload)%2 != 0 {
		data = append(data, 0)
	}
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	return data
}

func box(kind string, payload []byte) []byte {
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header, uint32(len(payload)+8))
	copy(header[4:], kind)
	return append(header, payload...)
}

func heicFixture(width, height uint32) []byte {
	ispe := make([]byte, 12)
	binary.BigEndian.PutUint32(ispe[4:], width)
	binary.BigEndian.PutUint32(ispe[8:], height)
	meta := box("meta", append(make([]byte, 4), box("iprp", box("ipco", box("ispe", ispe)))...))
	return append(box("ftyp", []byte("heic\x00\x00\x00\x00mif1heic")), meta...)
}
