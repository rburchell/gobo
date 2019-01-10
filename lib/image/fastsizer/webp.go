package fastsizer

func (f *decoder) getWEBPImageSize() (ImageSize, error) {
	slice, err := f.reader.Slice(26, 4)
	if err != nil {
		return ImageSize{}, err
	}

	return ImageSize{
		Width:  uint32(slice[1]&0x3f)<<8 | uint32(slice[0]),
		Height: uint32(slice[3]&0x3f)<<8 | uint32(slice[2]),
	}, nil
}
