package fastsizer

import (
	"io"
)

type xbuffer struct {
	r   io.Reader
	buf []byte
	off int
}

func (b *xbuffer) fill(end int) error {
	m := len(b.buf)
	if end > m {
		if end > cap(b.buf) {
			newcap := 1024
			for newcap < end {
				newcap *= 2
			}
			newbuf := make([]byte, end, newcap)
			copy(newbuf, b.buf)
			b.buf = newbuf
		} else {
			b.buf = b.buf[:end]
		}
		if n, err := io.ReadFull(b.r, b.buf[m:end]); err != nil {
			end = m + n
			b.buf = b.buf[:end]
			b.off = end
			return err
		}
	}
	b.off = end
	return nil
}

func (b *xbuffer) ReadAt(p []byte, off int64) (int, error) {
	o := int(off)
	end := o + len(p)
	if int64(end) != off+int64(len(p)) {
		return 0, io.ErrUnexpectedEOF
	}

	err := b.fill(end)
	return copy(p, b.buf[o:end]), err
}

func (b *xbuffer) Slice(off, n int) ([]byte, error) {
	end := off + n
	if err := b.fill(end); err != nil {
		return nil, err
	}
	return b.buf[off:end], nil
}

func (b *xbuffer) ReadByte() (byte, error) {
	current := b.off
	if err := b.fill(current + 1); err != nil {
		return 0, err
	}
	return b.buf[current], nil
}

func (b *xbuffer) ReadBytes(n int) ([]byte, error) {
	current := b.off
	if err := b.fill(current + n); err != nil {
		return nil, err
	}
	return b.buf[current : current+n], nil
}

func (b *xbuffer) ReadFull(p []byte) (int, error) {
	o := b.off
	end := o + len(p)

	err := b.fill(end)
	return copy(p, b.buf[o:end]), err
}

func newXbuffer(r io.Reader, sharedBuf []byte) *xbuffer {
	return &xbuffer{
		r:   r,
		buf: sharedBuf,
	}
}
