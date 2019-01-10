package fastsizer

import (
	"encoding/binary"
)

func (f *decoder) getPNGImageSize() (ImageSize, error) {
	slice, err := f.reader.Slice(16, 8)
	if err != nil {
		return ImageSize{}, err
	}

	return ImageSize{
		Width:  binary.BigEndian.Uint32(slice[0:4]),
		Height: binary.BigEndian.Uint32(slice[4:8]),
	}, nil
}
