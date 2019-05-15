package fastsizer

// ImageSize holds the width and height of an image
type ImageSize struct {
	Width  uint32
	Height uint32
}

type MirrorDirection int

const (
	MirrorNone MirrorDirection = iota
	MirrorHorizontal
	MirrorVertical
)

func (info ImageInfo) RotatedSize() ImageSize {
	switch info.Rotation {
	case 0:
		fallthrough
	case 180:
		return info.Size
	case 90:
		fallthrough
	case 270:
		return ImageSize{info.Size.Height, info.Size.Width}
	default:
		// Not supported..
		return info.Size
	}
}
