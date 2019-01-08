// +build !release

package engine

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// load an image using the native Go APIs
// this is quite slow at runtime, but fast to build, so we only use it for
// developer (non-release) builds.
func LoadImage(file string) (*image.RGBA, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("LoadImage: couldn't open %s: %s", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, fmt.Errorf("LoadImage: couldn't decode %s: %s", file, err)
	}

	// ### convert to rgba so we give a consistent output
	// ### should only do this if the image is not already rgba format (from decode).
	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride %d", rgba.Stride)
	}

	draw.Draw(rgba, rgba.Bounds(), img, image.ZP, draw.Src)
	return rgba, nil
}
