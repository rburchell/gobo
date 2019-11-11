package fastsizer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

func (f *decoder) getJPEGInfo(info *ImageInfo) error {
	offset := 2
	var err error
	tmp := make([]byte, 2)
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

type ifdType int

const (
	primaryIfd ifdType = iota
	exifIfd
	gpsIfd
)

type tagToField struct {
	id    uint16
	name  string
	field interface{}
}

const exifDebug = false

func (f *decoder) readIfd(ifdType ifdType, r io.Reader, byteOrder binary.ByteOrder, exif *ExifData, offs int) error {
	primaryTags := []tagToField{
		tagToField{
			id:    0x010e,
			name:  "ImageDescription",
			field: &exif.ImageDescription,
		},
		tagToField{
			id:    0x010f,
			name:  "Make",
			field: &exif.Make,
		},
		tagToField{
			id:    0x0110,
			name:  "Model",
			field: &exif.Model,
		},
		tagToField{
			id:    0x0112,
			name:  "Orientation",
			field: &exif.Orientation,
		},
		tagToField{
			id:    0x011A,
			name:  "XResolution",
			field: nil,
		},
		tagToField{
			id:    0x011B,
			name:  "YResolution",
			field: nil,
		},
		tagToField{
			id:    0x0128,
			name:  "ResolutionUnit",
			field: nil,
		},
		tagToField{
			id:    0x0131,
			name:  "Software",
			field: &exif.Software,
		},
		tagToField{
			id:    0x0132,
			name:  "DateTime",
			field: &exif.DateTime,
		},
		tagToField{
			id:    0x013B,
			name:  "Artist",
			field: &exif.Artist,
		},
		tagToField{
			id:    0x013C,
			name:  "HostComputer",
			field: &exif.HostComputer,
		},
		tagToField{
			id:    0x1001,
			name:  "RelatedImageWidth",
			field: nil,
		},
		tagToField{
			id:    0x1002,
			name:  "RelatedImageHeight",
			field: nil,
		},
		tagToField{
			id:    0x0211,
			name:  "YCbCrCoefficients",
			field: nil,
		},
		tagToField{
			id:    0x0213,
			name:  "YCbCrPositioning",
			field: nil,
		},
		tagToField{
			id:    0xA401,
			name:  "CustomRendered",
			field: nil,
		},
		tagToField{
			id:    0xA402,
			name:  "ExposureMode",
			field: nil,
		},
		tagToField{
			id:    0xA403,
			name:  "WhiteBalance",
			field: nil,
		},
		tagToField{
			id:    0xA404,
			name:  "DigitalZoomRatio",
			field: nil,
		},
		tagToField{
			id:    0xA405,
			name:  "FocalLengthIn35mmFilm",
			field: nil,
		},
		tagToField{
			id:    0xA406,
			name:  "SceneCaptureType",
			field: nil,
		},
		tagToField{
			id:    0xA407,
			name:  "GainControl",
			field: nil,
		},
		tagToField{
			id:    0xA408,
			name:  "Contrast",
			field: nil,
		},
		tagToField{
			id:    0xA409,
			name:  "Saturation",
			field: nil,
		},
		tagToField{
			id:    0xA40A,
			name:  "Sharpness",
			field: nil,
		},
		tagToField{
			id:    0xC4A5,
			name:  "PrintImageMatching",
			field: nil,
		},
		tagToField{
			id:    0x8298,
			name:  "Copyright",
			field: &exif.Copyright,
		},
		tagToField{
			id:    0x8769,
			name:  "ExifOffset",
			field: nil,
		},
	}

	exifTags := []tagToField{}

	// Read the number of tags.
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return err
	}

	// Parse the tags.
	str := ""
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return err
		}
		var fieldType uint16
		if err := binary.Read(r, byteOrder, &fieldType); err != nil {
			return err
		}
		var componentCount uint32
		if err := binary.Read(r, byteOrder, &componentCount); err != nil {
			return err
		}
		var data uint32
		if err := binary.Read(r, byteOrder, &data); err != nil {
			return err
		}

		tagList := primaryTags
		switch ifdType {
		case primaryIfd:
			tagList = primaryTags
		case exifIfd:
			tagList = exifTags
		default:
			log.Printf("Unknown ifd type %d", ifdType)
		}

		found := false
		for _, field := range tagList {
			if field.id == tag {
				switch tf := field.field.(type) {
				case *string:
					err := f.readStringTag(&str, offs, componentCount, data)
					if err != nil {
						return err
					}
					*tf = str
					if exifDebug {
						log.Printf("Parsed string field %x (%s) as %v", field.id, field.name, field.field)
					}
				case *ExifOrientation:
					if data < 0 || data > 8 {
						return fmt.Errorf("invalid orientation tag (%d)", data)
					}

					*tf = ExifOrientation(data)
					if exifDebug {
						log.Printf("Parsed orientation field %x (%s) as %v", field.id, field.name, field.field)
					}
				case nil:
					if ifdType == primaryIfd && field.id == 0x8769 {
						/*
							log.Printf("Offset to another ifd")
							err := f.readIfd(exifIfd, r, byteOrder, exif, offs)
							if err != nil {
								return err
							}
							continue
						*/
					}
					if exifDebug {
						log.Printf("Ignored field %s (%d %d)", field.name, componentCount, data)
					}
				default:
					log.Printf("unknown exif field type %T", field.field)
				}
				found = true
				break
			}
		}

		if found == false {
			log.Printf("Unknown EXIF tag: %x %d %d %d", tag, fieldType, componentCount, data)
		}
	}

	return nil
}

func (f *decoder) readStringTag(place *string, offs int, componentCount, data uint32) error {
	tagDataOffset := offs      /* section offset */
	tagDataOffset += int(data) /* tag data offset */
	tagDataOffset += 6         /* not sure why, but this seems to be necessary... */
	tmp, err := f.reader.Slice(tagDataOffset, int(componentCount))
	if err != nil {
		return err
	}

	*place = string(tmp)
	return nil
}

var errNotExif = errors.New("not exif")

// Adapted from https://github.com/disintegration/imageorient/
func (f *decoder) readExif(info *ImageInfo, offs, n int) error {
	buf, err := f.reader.Slice(offs, n)
	if err != nil {
		return err
	}
	r := bytes.NewBuffer(buf)

	// Check if EXIF header is present.
	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return err
	}
	const exifHeader = 0x45786966
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

	const (
		byteOrderBE = 0x4d4d
		byteOrderLE = 0x4949
	)

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

	err = f.readIfd(primaryIfd, r, byteOrder, &info.ExifData, offs)

	return err
}
