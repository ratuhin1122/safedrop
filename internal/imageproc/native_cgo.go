//go:build cgo && !windows

package imageproc

/*
#cgo pkg-config: vips
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"context"
	"errors"
	"math"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"github.com/davidbyttow/govips/v2/vips"
)

var startup sync.Once
var startupErr error

type fileEncoder struct{}

func nativeEncoder() (Encoder, error) {
	startup.Do(func() {
		startupErr = vips.Startup(&vips.Config{ConcurrencyLevel: 1, MaxCacheMem: 32 << 20, MaxCacheSize: 0})
	})
	return fileEncoder{}, startupErr
}

func (fileEncoder) Encode(ctx context.Context, input, output string, width int, avif bool, maxPixels uint64) (Dimensions, error) {
	if err := ctx.Err(); err != nil {
		return Dimensions{}, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	source, target := C.CString(input), C.CString(output)
	defer C.free(unsafe.Pointer(source))
	defer C.free(unsafe.Pointer(target))
	var w, h C.int
	if C.sd_probe(source, &w, &h) != 0 {
		return Dimensions{}, nativeError()
	}
	if w <= 0 || h <= 0 || uint64(w) > maxPixels/uint64(h) {
		return Dimensions{}, errors.New("image exceeds maximum pixels")
	}
	height := math.Max(1, math.Round(float64(h)*float64(width)/float64(w)))
	if height > math.MaxInt32 || uint64(width) > maxPixels/uint64(height) {
		return Dimensions{}, errors.New("variant exceeds maximum pixels")
	}
	params := vips.NewImportParams()
	params.AutoRotate.Set(true)
	preview, err := vips.LoadThumbnailFromFile(input, width, int(height), vips.InterestingNone, vips.SizeForce, params)
	if err != nil {
		return Dimensions{}, err
	}
	defer preview.Close()
	if err := preview.RemoveMetadata(); err != nil {
		return Dimensions{}, err
	}
	dimensions := Dimensions{int32(preview.Width()), int32(preview.Height())}
	if avif {
		if C.sd_encode(source, target, C.int(width), C.int(height), 1) != 0 {
			return Dimensions{}, nativeError()
		}
	} else {
		file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return Dimensions{}, err
		}
		params := vips.NewWebpExportParams()
		params.StripMetadata, params.Quality = true, 80
		err = preview.SaveToWriterWebp(file, params)
		if err := errors.Join(err, file.Close()); err != nil {
			return Dimensions{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Dimensions{}, err
	}
	return dimensions, nil
}

func nativeError() error {
	err := errors.New(C.GoString(C.vips_error_buffer()))
	C.vips_error_clear()
	return err
}
