package validate

import (
	"encoding/binary"
	"net/http"
)

func detectType(header []byte) string {
	if heicBrand(header) {
		return "image/heic"
	}
	mime := http.DetectContentType(header)
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
		return mime
	}
	return ""
}

// HEIC uses an ISO BMFF file-type box; generic mif1/avif brands are insufficient.
// Only examine brands that occur within the first 512 bytes of the upload.
func heicBrand(header []byte) bool {
	if len(header) < 16 || string(header[4:8]) != "ftyp" {
		return false
	}
	size, start := uint64(binary.BigEndian.Uint32(header[:4])), 8
	if size == 1 {
		if len(header) < 24 {
			return false
		}
		size, start = binary.BigEndian.Uint64(header[8:16]), 16
	}
	if size < uint64(start+8) || (size-uint64(start))%4 != 0 {
		return false
	}
	end := len(header)
	if size < uint64(end) {
		end = int(size)
	}
	for offset := start; offset+4 <= end; offset += 4 {
		if offset == start+4 {
			continue
		} // minor version, not a brand
		switch string(header[offset : offset+4]) {
		case "heic", "heix", "hevc", "hevx":
			return true
		}
	}
	return false
}
