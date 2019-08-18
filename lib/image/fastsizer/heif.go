package fastsizer

import (
	"errors"

	"go4.org/media/heif"
)

func (d *decoder) getHEIFImageSize() (ImageSize, error) {
	file := heif.Open(d.reader)
	it, err := file.PrimaryItem()
	if err != nil {
		return ImageSize{}, err
	}

	w, h, ok := it.VisualDimensions()
	if !ok {
		return ImageSize{}, errors.New("cannot read heif dimensions")
	}

	return ImageSize{Width: uint32(w), Height: uint32(h)}, nil
}
