package fastsizer

import (
	"fmt"
	"io"
)

type ImageInfo struct {
	Size     ImageSize
	Type     ImageType
	Rotation int
	Mirror   MirrorDirection
}

// FastImage instance needs to be initialized before use
type FastImage struct {
	tb             []byte
	internalBuffer []byte
}

// NewFastSizer returns a FastImage client
func NewFastSizer() *FastImage {
	return &FastImage{tb: make([]byte, 2), internalBuffer: make([]byte, 0, 2)}
}

type decoder struct {
	reader  *xbuffer
	minimal bool
}

//Detect image type and size
func (this *FastImage) Detect(reader io.Reader) (ImageType, ImageSize, error) {
	this.internalBuffer = this.internalBuffer[0:0]
	d := &decoder{
		reader:  newXbuffer(reader, this.internalBuffer),
		minimal: true,
	}
	info, err := this.detectInternal(d, reader)
	return info.Type, info.Size, err
}

func (this *FastImage) DetectInfo(reader io.Reader) (ImageInfo, error) {
	this.internalBuffer = this.internalBuffer[0:0]
	d := &decoder{reader: newXbuffer(reader, this.internalBuffer)}
	return this.detectInternal(d, reader)
}

func (this *FastImage) detectInternal(d *decoder, reader io.Reader) (ImageInfo, error) {
	var info ImageInfo
	var e error

	if _, err := d.reader.ReadAt(this.tb, 0); err != nil {
		return info, err
	}

	ok := false

	switch this.tb[0] {
	case 'B':
		switch this.tb[1] {
		case 'M':
			info.Type = BMP
			info.Size, e = d.getBMPImageSize()
			ok = true
		}
	case 0x47:
		switch this.tb[1] {
		case 0x49:
			info.Type = GIF
			info.Size, e = d.getGIFImageSize()
			ok = true
		}
	case 0xFF:
		switch this.tb[1] {
		case 0xD8:
			info.Type = JPEG
			e = d.getJPEGInfo(&info)
			ok = true
		}
	case 0x89:
		switch this.tb[1] {
		case 0x50:
			info.Type = PNG
			info.Size, e = d.getPNGImageSize()
			ok = true
		}
	case 'I':
		switch this.tb[1] {
		case 'I':
			info.Type = TIFF
			info.Size, e = d.getTIFFImageSize()
			ok = true
		}
	case 'M':
		switch this.tb[1] {
		case 'M':
			info.Type = TIFF
			info.Size, e = d.getTIFFImageSize()
			ok = true
		}
	case 'R':
		switch this.tb[1] {
		case 'I':
			info.Type = WEBP
			info.Size, e = d.getWEBPImageSize()
			ok = true
		}
	}

	this.internalBuffer = d.reader.buf

	if !ok {
		return info, fmt.Errorf("Unknown image type (%v)", this.tb)
	}
	return info, e
}
