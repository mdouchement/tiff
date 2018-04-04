package tiff

import (
	"compress/zlib"
	"fmt"
	"image"
	"io"
	"io/ioutil"

	"golang.org/x/image/tiff/lzw"

	"github.com/mdouchement/hdr/hdrcolor"
)

type decoder struct {
	*idf
	config image.Config
	mode   imageMode
	bpp    uint

	buf   []byte
	off   int    // Current offset in buf.
	v     uint32 // Buffer value for reading with arbitrary bit depths.
	nbits uint   // Remaining number of bits in v.
}

func newDecoder(r io.Reader) (*decoder, error) {
	idf, err := newIDF(newReaderAt(r))
	if err != nil {
		return nil, err
	}

	d := &decoder{
		idf: idf,
	}

	d.config.Width = int(d.firstVal(tImageWidth))
	d.config.Height = int(d.firstVal(tImageLength))

	if _, ok := d.features[tBitsPerSample]; !ok {
		return nil, FormatError("BitsPerSample tag missing")
	}
	d.bpp = d.firstVal(tBitsPerSample)

	// Determine the image mode.
	switch d.firstVal(tPhotometricInterpretation) {
	case pWhiteIsZero:
		fallthrough
	case pBlackIsZero:
		fallthrough
	case pPaletted:
		fallthrough
	case pTransMask:
		fallthrough
	case pCMYK:
		// All LDR modes are droped.
		return nil, UnsupportedError("color model, use Golang's lib for LDR images")
	case pRGB:
		d.mode = mRGB
		d.config.ColorModel = hdrcolor.RGBModel
	case pLogL:
		d.mode = mLogL
		d.config.ColorModel = hdrcolor.XYZModel
	case pLogLuv:
		d.mode = mLogLuv
		d.config.ColorModel = hdrcolor.XYZModel
	default:
		return nil, UnsupportedError("color model")
	}

	return d, nil
}

// readBits reads n bits from the internal buffer starting at the current offset.
func (d *decoder) readBits(n uint) uint32 {
	for d.nbits < n {
		d.v <<= 8
		d.v |= uint32(d.buf[d.off])
		d.off++
		d.nbits += 8
	}
	d.nbits -= n
	rv := d.v >> d.nbits
	d.v &^= rv << d.nbits
	return rv
}

// flushBits discards the unread bits in the buffer used by readBits.
// It is used at the end of a line.
func (d *decoder) flushBits() {
	d.v = 0
	d.nbits = 0
}

// decompress decompress a Strip.
func (d *decoder) decompress(offset, n int64, blockWidth, blockHeight int) (err error) {
	switch d.firstVal(tCompression) {
	// According to the spec, Compression does not have a default value,
	// but some tools interpret a missing Compression value as none so we do
	// the same.
	case cNone, 0:
		if b, ok := d.r.(*buffer); ok {
			d.buf, err = b.Slice(int(offset), int(n))
		} else {
			d.buf = make([]byte, n)
			_, err = d.r.ReadAt(d.buf, offset)
		}
	case cLZW:
		r := lzw.NewReader(io.NewSectionReader(d.r, offset, n), lzw.MSB, 8)
		d.buf, err = ioutil.ReadAll(r)
		r.Close()
	case cDeflate, cDeflateOld:
		var r io.ReadCloser
		r, err = zlib.NewReader(io.NewSectionReader(d.r, offset, n))
		if err != nil {
			return
		}
		d.buf, err = ioutil.ReadAll(r)
		r.Close()
	case cPackBits:
		d.buf, err = unpackBits(io.NewSectionReader(d.r, offset, n))
	case cSGILogRLE:
		d.buf, err = unRLE(io.NewSectionReader(d.r, offset, n), d.mode, blockWidth, blockHeight)
	default:
		err = UnsupportedError(fmt.Sprintf("compression value %d", d.firstVal(tCompression)))
	}
	return
}
