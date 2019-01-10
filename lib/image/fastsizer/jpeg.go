package fastsizer

import (
	"fmt"
)

func (f *decoder) getJPEGImageSize() (ImageSize, error) {
	offset := 2
	var err error
	tmp := make([]byte, 2)
	for {
		tmp, err = f.reader.Slice(offset, 2)
		if err != nil {
			return ImageSize{}, err
		}
		offset += 2
		for tmp[0] != 0xff {
			tmp[0] = tmp[1]
			tmp[1], err = f.reader.ReadByte()
			if err != nil {
				return ImageSize{}, err
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
				return ImageSize{}, err
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
			return ImageSize{}, err
		}
		offset += 2
		n := int(tmp[0])<<8 + int(tmp[1]) - 2
		if n < 0 {
			return ImageSize{}, fmt.Errorf("short segment length")
		}
		switch marker {
		case sof0Marker, sof1Marker, sof2Marker:
			tmp, err = f.reader.Slice(offset, n)
			if err == nil {
				if tmp[0] != 8 {
					err = fmt.Errorf("only support 8-bit precision")
					return ImageSize{}, err
				} else {
					return ImageSize{
						Width:  uint32(int(tmp[3])<<8 + int(tmp[4])),
						Height: uint32(int(tmp[1])<<8 + int(tmp[2])),
					}, nil
				}
			}
		case dhtMarker, dqtMarker, driMarker, app0Marker, app14Marker:
			offset += n
		case sosMarker:
			return ImageSize{}, fmt.Errorf("meet sos marker")
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
			return ImageSize{}, err
		}
	}
	return ImageSize{}, fmt.Errorf("fail get size")
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
	app14Marker = 0xee
	app15Marker = 0xef
)
