package fastsizer

import (
	"fmt"
)

// ImageSize holds the width and height of an image
type ImageSize struct {
	Width  uint32
	Height uint32
}

// How many degrees the image is rotated.
// 0 is not rotated.
type RotationDegrees int

func (this RotationDegrees) String() string {
	return fmt.Sprintf("%d degrees", int(this))
}

type MirrorDirection int

const (
	MirrorNone MirrorDirection = iota
	MirrorHorizontal
	MirrorVertical
)

func (this MirrorDirection) String() string {
	switch this {
	case MirrorNone:
		return "Not mirrored"
	case MirrorHorizontal:
		return "Horizontal mirror"
	case MirrorVertical:
		return "Vertical mirror"
	}
	return "Not mirrored"
}

// The orientation of the image, retrieved from EXIF data.
type ExifOrientation int

// The degree of rotation of the given orientation.
func (this ExifOrientation) Rotation() RotationDegrees {
	switch this {
	case 2:
	case 3:
		return 180
	case 4:
	case 5:
		fallthrough
	case 8:
		return 270
	case 7:
		fallthrough
	case 6:
		return 90
	}
	return 0
}

// The direction the orientation is mirrored.
func (this ExifOrientation) MirrorDirection() MirrorDirection {
	switch this {
	case 2:
		return MirrorHorizontal
	case 3:
	case 4:
		return MirrorVertical
	case 5:
		return MirrorHorizontal
	case 8:
	case 7:
		return MirrorHorizontal
	case 6:
	}
	return MirrorNone
}

func (this ExifOrientation) String() string {
	return fmt.Sprintf("%s, %s", this.Rotation(), this.MirrorDirection())
}

func (info ImageInfo) RotatedSize() ImageSize {
	switch info.ExifData.Orientation.Rotation() {
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
