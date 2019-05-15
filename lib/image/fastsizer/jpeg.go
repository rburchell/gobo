package fastsizer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

func (f *decoder) getJPEGInfo(info *ImageInfo) error {
	offset := 2
	var err error
	tmp := make([]byte, 2)
	for {
		tmp, err = f.reader.Slice(offset, 2)
		if err != nil {
			return err
		}
		offset += 2
		for tmp[0] != 0xff {
			tmp[0] = tmp[1]
			tmp[1], err = f.reader.ReadByte()
			if err != nil {
				return err
			}
			offset++
		}
		marker := tmp[1]
		if marker == 0 {
			continue
		}
		for marker == 0xff {
			marker, err = f.reader.ReadByte()
			if err != nil {
				return err
			}
			offset++
		}
		if marker == eoiMarker {
			break
		}
		if rst0Marker <= marker && marker <= rst7Marker {
			continue
		}
		_, err = f.reader.ReadFull(tmp)
		if err != nil {
			return err
		}
		offset += 2
		n := int(tmp[0])<<8 + int(tmp[1]) - 2
		if n < 0 {
			return fmt.Errorf("short segment length")
		}
		switch marker {
		case sof0Marker, sof1Marker, sof2Marker:
			tmp, err = f.reader.Slice(offset, n)
			if err == nil {
				if tmp[0] != 8 {
					err = fmt.Errorf("only support 8-bit precision")
					return err
				} else {
					info.Size = ImageSize{
						Width:  uint32(int(tmp[3])<<8 + int(tmp[4])),
						Height: uint32(int(tmp[1])<<8 + int(tmp[2])),
					}
					return nil
				}
			}
		case dhtMarker, dqtMarker, driMarker, app0Marker, app14Marker:
			offset += n
		case sosMarker:
			return fmt.Errorf("meet sos marker")
		case app1Marker:
			if !f.minimal {
				if err := f.readExif(info, offset, n); err != nil && err != errNotExif {
					return err
				}
			}
			offset += n
		default:
			if app0Marker <= marker && marker <= app15Marker || marker == comMarker {
				offset += n
			} else if marker < 0xc0 {
				err = fmt.Errorf("unknown marker")
			} else {
				err = fmt.Errorf("unsupport marker")
			}
		}
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("fail get size")
}

var errNotExif = errors.New("not exif")

// Adapted from https://github.com/disintegration/imageorient/
func (f *decoder) readExif(info *ImageInfo, offs, n int) error {
	// EXIF marker
	const (
		markerSOI      = 0xffd8
		markerAPP1     = 0xffe1
		exifHeader     = 0x45786966
		byteOrderBE    = 0x4d4d
		byteOrderLE    = 0x4949
		orientationTag = 0x0112
	)

	// XXX This is a lazy way to avoid rewriting the logic
	buf := make([]byte, n)
	if _, err := f.reader.ReadFull(buf); err != nil {
		return err
	}
	r := bytes.NewBuffer(buf)

	// Check if EXIF header is present.
	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return err
	}
	if header != exifHeader {
		return errNotExif
	}
	if _, err := io.CopyN(ioutil.Discard, r, 2); err != nil {
		return err
	}

	// Read byte order information.
	var (
		byteOrderTag uint16
		byteOrder    binary.ByteOrder
	)
	if err := binary.Read(r, binary.BigEndian, &byteOrderTag); err != nil {
		return err
	}
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return errors.New("invalid byte order")
	}
	if _, err := io.CopyN(ioutil.Discard, r, 2); err != nil {
		return err
	}

	// Skip the EXIF offset.
	var offset uint32
	if err := binary.Read(r, byteOrder, &offset); err != nil {
		return err
	}
	if offset < 8 {
		return errors.New("invalid EXIF offset")
	}
	if _, err := io.CopyN(ioutil.Discard, r, int64(offset-8)); err != nil {
		return err
	}

	// Read the number of tags.
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return err
	}

	// Find the orientation tag.
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return err
		}
		if tag != orientationTag {
			if _, err := io.CopyN(ioutil.Discard, r, 10); err != nil {
				return err
			}
			continue
		}
		if _, err := io.CopyN(ioutil.Discard, r, 6); err != nil {
			return err
		}
		var val uint16
		if err := binary.Read(r, byteOrder, &val); err != nil {
			return err
		}
		if val < 0 || val > 8 {
			return fmt.Errorf("invalid orientation tag (%d)", val)
		}

		switch val {
		case 2:
			info.Mirror = MirrorHorizontal
		case 3:
			info.Rotation = 180
		case 4:
			info.Mirror = MirrorVertical
		case 5:
			info.Mirror = MirrorHorizontal
			fallthrough
		case 8:
			info.Rotation = 270
		case 7:
			info.Mirror = MirrorHorizontal
			fallthrough
		case 6:
			info.Rotation = 90
		}

		return nil
	}

	return nil
}

const (
	sof0Marker = 0xc0 // Start Of Frame (Baseline).
	sof1Marker = 0xc1 // Start Of Frame (Extended Sequential).
	sof2Marker = 0xc2 // Start Of Frame (Progressive).
	dhtMarker  = 0xc4 // Define Huffman Table.
	rst0Marker = 0xd0 // ReSTart (0).
	rst7Marker = 0xd7 // ReSTart (7).
	soiMarker  = 0xd8 // Start Of Image.
	eoiMarker  = 0xd9 // End Of Image.
	sosMarker  = 0xda // Start Of Scan.
	dqtMarker  = 0xdb // Define Quantization Table.
	driMarker  = 0xdd // Define Restart Interval.
	comMarker  = 0xfe // COMment.
	// "APPlication specific" markers aren't part of the JPEG spec per se,
	// but in practice, their use is described at
	// http://www.sno.phy.queensu.ca/~phil/exiftool/TagNames/JPEG.html
	app0Marker  = 0xe0
	app1Marker  = 0xe1
	app14Marker = 0xee
	app15Marker = 0xef
)
