//go:build !cgo || windows

package imageproc

func nativeEncoder() (Encoder, error) { return nil, ErrNativeUnavailable }
