package fastsizer

func (f *decoder) getGIFImageSize() (ImageSize, error) {
	slice, err := f.reader.Slice(6, 4)
	if err != nil {
		return ImageSize{}, err
	}

	return ImageSize{
		Width:  uint32(readULint16(slice[0:2])),
		Height: uint32(readULint16(slice[2:4])),
	}, nil
}
