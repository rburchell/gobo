package fastsizer

import (
	"fmt"
	"io"
)

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
	reader *xbuffer
}

//Detect image type and size
func (this *FastImage) Detect(reader io.Reader) (ImageType, ImageSize, error) {
	this.internalBuffer = this.internalBuffer[0:0]
	d := &decoder{reader: newXbuffer(reader, this.internalBuffer)}

	var t ImageType
	var s ImageSize
	var e error

	if _, err := d.reader.ReadAt(this.tb, 0); err != nil {
		return Unknown, ImageSize{}, err
	}

	ok := false

	switch this.tb[0] {
	case 'B':
		switch this.tb[1] {
		case 'M':
			t = BMP
			s, e = d.getBMPImageSize()
			ok = true
		}
	case 0x47:
		switch this.tb[1] {
		case 0x49:
			t = GIF
			s, e = d.getGIFImageSize()
			ok = true
		}
	case 0xFF:
		switch this.tb[1] {
		case 0xD8:
			t = JPEG
			s, e = d.getJPEGImageSize()
			ok = true
		}
	case 0x89:
		switch this.tb[1] {
		case 0x50:
			t = PNG
			s, e = d.getPNGImageSize()
			ok = true
		}
	case 'I':
		switch this.tb[1] {
		case 'I':
			t = TIFF
			s, e = d.getTIFFImageSize()
			ok = true
		}
	case 'M':
		switch this.tb[1] {
		case 'M':
			t = TIFF
			s, e = d.getTIFFImageSize()
			ok = true
		}
	case 'R':
		switch this.tb[1] {
		case 'I':
			t = WEBP
			s, e = d.getWEBPImageSize()
			ok = true
		}
	}

	this.internalBuffer = d.reader.buf

	if !ok {
		return Unknown, ImageSize{}, fmt.Errorf("Unknown image type (%v)", this.tb)
	}
	return t, s, e
}
