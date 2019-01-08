// +build release

package engine

import (
	"fmt"
	"image"
	"unsafe"
)

// #cgo LDFLAGS: -lm
// #define STB_IMAGE_IMPLEMENTATION
// #include "stb_image.h"
// #include <stdlib.h>
import "C"

func LoadImage(file string) (*image.RGBA, error) {
	var x, y, n C.int
	cfn := C.CString(file)
	defer C.free(unsafe.Pointer(cfn))
	data := C.stbi_load(cfn, &x, &y, &n, 4)
	godata := C.GoBytes(unsafe.Pointer(data), y*x*4)
	rgba := &image.RGBA{
		Pix:    godata,
		Stride: 4,
		Rect:   image.Rect(0, 0, int(x), int(y)),
	}
	C.stbi_image_free(unsafe.Pointer(data))

	if rgba == nil {
		// ### can we get an error?
		return nil, fmt.Errorf("can't load %s", file)
	}
	return rgba, nil
}
