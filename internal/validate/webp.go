package validate

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

func init() {
	image.RegisterFormat("webp", "RIFF????WEBP", noPixelDecoder, webpConfig)
	image.RegisterFormat("heic", "????ftyp", noPixelDecoder, heicConfig)
}

// These registrations intentionally provide configuration decoding only.
func noPixelDecoder(io.Reader) (image.Image, error) { return nil, image.ErrFormat }

func webpConfig(r io.Reader) (image.Config, error) {
	var header [20]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return image.Config{}, err
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WEBP" {
		return image.Config{}, image.ErrFormat
	}
	riffSize := uint64(binary.LittleEndian.Uint32(header[4:8]))
	chunkSize := uint64(binary.LittleEndian.Uint32(header[16:20]))
	if riffSize < 12 || chunkSize+(chunkSize%2) > riffSize-12 {
		return image.Config{}, image.ErrFormat
	}
	var payload [10]byte
	length := 10
	if string(header[12:16]) == "VP8L" {
		length = 5
	}
	if chunkSize < uint64(length) {
		return image.Config{}, image.ErrFormat
	}
	if _, err := io.ReadFull(r, payload[:length]); err != nil {
		return image.Config{}, err
	}
	var width, height uint32
	switch string(header[12:16]) {
	case "VP8X":
		if chunkSize != 10 {
			return image.Config{}, image.ErrFormat
		}
		width, height = little24(payload[4:7])+1, little24(payload[7:10])+1
		if uint64(width)*uint64(height) > 1<<32-1 {
			return image.Config{}, image.ErrFormat
		}
	case "VP8L":
		if payload[0] != 0x2f || payload[4]>>5 != 0 {
			return image.Config{}, image.ErrFormat
		}
		bits := binary.LittleEndian.Uint32(payload[1:5])
		width, height = (bits&0x3fff)+1, ((bits>>14)&0x3fff)+1
	case "VP8 ":
		if payload[0]&1 != 0 || string(payload[3:6]) != "\x9d\x01\x2a" {
			return image.Config{}, image.ErrFormat
		}
		width = uint32(binary.LittleEndian.Uint16(payload[6:8]) & 0x3fff)
		height = uint32(binary.LittleEndian.Uint16(payload[8:10]) & 0x3fff)
	default:
		return image.Config{}, image.ErrFormat
	}
	if width == 0 || height == 0 {
		return image.Config{}, image.ErrFormat
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: int(width), Height: int(height)}, nil
}

func little24(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 }
