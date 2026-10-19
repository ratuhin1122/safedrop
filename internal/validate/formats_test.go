package validate

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"
)

func TestTypeSniffing(t *testing.T) {
	compatible := heicFixture(2, 3)
	copy(compatible[8:12], "mif1")
	for _, tt := range []struct {
		name string
		data []byte
		want string
	}{
		{"compatible HEIC brand", compatible, "image/heic"},
		{"AVIF excluded", box("ftyp", []byte("avif\x00\x00\x00\x00mif1avif")), ""},
		{"generic HEIF excluded", box("ftyp", []byte("mif1\x00\x00\x00\x00mif1")), ""},
		{"brand outside ftyp", append(box("ftyp", []byte("mif1\x00\x00\x00\x00")), []byte("heic")...), ""},
		{"plain text HEIC", []byte("heic file"), ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectType(tt.data); got != tt.want {
				t.Fatalf("type = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMalformedImageConfigurations(t *testing.T) {
	badSize := heicFixture(2, 3)
	binary.BigEndian.PutUint32(badSize[24:28], 4) // meta size smaller than its header
	badRIFF := webpFixture("VP8X", 2, 3)
	binary.LittleEndian.PutUint32(badRIFF[4:8], 4)
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{"HEIC zero extent", heicFixture(0, 3)},
		{"HEIC malformed box size", badSize},
		{"HEIC no image properties", box("ftyp", []byte("heic\x00\x00\x00\x00"))},
		{"HEIC root ispe is not an image property", append(box("ftyp", []byte("heic\x00\x00\x00\x00")), box("ispe", make([]byte, 12))...)},
		{"WebP inconsistent RIFF size", badRIFF},
		{"WebP unknown image chunk", webpFixture("NONE", 2, 3)},
		{"WebP zero lossy width", webpFixture("VP8 ", 0, 3)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if cfg, _, err := image.DecodeConfig(bytes.NewReader(tt.data)); err == nil {
				t.Fatalf("accepted malformed configuration: %+v", cfg)
			}
		})
	}
}

func TestTruncatedHEIC(t *testing.T) {
	data := heicFixture(2, 3)
	for n := 0; n < len(data); n++ {
		if _, _, err := image.DecodeConfig(bytes.NewReader(data[:n])); err == nil {
			t.Fatalf("accepted truncation at byte %d", n)
		}
	}
}
