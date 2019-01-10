package fastsizer

func (f *decoder) getBMPImageSize() (ImageSize, error) {
	slice, err := f.reader.Slice(18, 8)
	if err != nil {
		return ImageSize{}, err
	}

	return ImageSize{
		Width:  uint32(readUint32(slice[0:4])),
		Height: uint32(readUint32(slice[4:8])),
	}, nil
}
