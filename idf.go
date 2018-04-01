package tiff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

//------------------------//
// Header parser          //
//------------------------//

type idf struct {
	r         io.ReaderAt
	byteOrder binary.ByteOrder
	features  map[uint16][]uint
}

func newIDF(r io.ReaderAt) (d *idf, err error) {
	d = &idf{
		r:        r,
		features: make(map[uint16][]uint),
	}

	p := make([]byte, 8)
	if _, err := d.r.ReadAt(p, 0); err != nil {
		return nil, err
	}
	switch string(p[0:4]) {
	case leHeader:
		d.byteOrder = binary.LittleEndian
	case beHeader:
		d.byteOrder = binary.BigEndian
	default:
		return nil, FormatError("malformed header")
	}

	ifdOffset := int64(d.byteOrder.Uint32(p[4:8]))

	// The first two bytes contain the number of entries (12 bytes each).
	if _, err := d.r.ReadAt(p[0:2], ifdOffset); err != nil {
		return nil, err
	}
	numItems := int(d.byteOrder.Uint16(p[0:2]))

	// All IFD entries are read in one chunk.
	p = make([]byte, ifdLen*numItems)
	if _, err := d.r.ReadAt(p, ifdOffset+2); err != nil {
		return nil, err
	}

	for i := 0; i < len(p); i += ifdLen {
		if err := d.parseIFD(p[i : i+ifdLen]); err != nil {
			return nil, err
		}
	}

	return
}

func (d *idf) String() string {
	buf := bytes.NewBufferString("")
	for idf, v := range d.features {
		buf.WriteString(fmt.Sprintf("%s: %s\n", tagname(idf), valuename(idf, v)))
	}
	buf.WriteString(fmt.Sprintf("ByteOrder: %v\n", d.byteOrder))
	buf.WriteString(fmt.Sprintf("BPP: %d\n", d.firstVal(tBitsPerSample)))
	buf.WriteString(fmt.Sprintf("Bounds: %dx%d\n", d.firstVal(tImageWidth), d.firstVal(tImageLength)))
	return buf.String()
}

// firstVal returns the first uint of the features entry with the given tag,
// or 0 if the tag does not exist.
func (d *idf) firstVal(tag uint16) uint {
	f := d.features[tag]
	if len(f) == 0 {
		return 0
	}
	return f[0]
}

// firstDouble returns the first float64 of the features entry with the given tag,
// or 0 if the tag does not exist.
func (d *idf) firstDouble(tag uint16) float64 {
	f := d.features[tag]
	if len(f) == 0 {
		return 0
	}
	return math.Float64frombits(uint64(f[0]))
}

// parseIFD decides whether the the IFD entry in p is "interesting" and
// stows away the data in the decoder.
func (d *idf) parseIFD(p []byte) error {
	tag := d.byteOrder.Uint16(p[0:2])
	switch tag {
	case tBitsPerSample,
		tExtraSamples,
		tPhotometricInterpretation,
		tCompression,
		tPredictor,
		tStripOffsets,
		tStripByteCounts,
		tSamplesPerPixel,
		tRowsPerStrip,
		tTileWidth,
		tTileLength,
		tTileOffsets,
		tTileByteCounts,
		tPlanarConfiguration,
		tImageLength,
		tImageWidth,
		tStonits:
		val, err := d.ifdUint(p)
		if err != nil {
			return err
		}
		// fmt.Println(tagname(int(tag)), "-", val)
		d.features[tag] = val
	case tSampleFormat:
		// Page 27 of the spec: If the SampleFormat is present and
		// the value is not 1 [= unsigned integer data], a Baseline
		// TIFF reader that cannot handle the SampleFormat value
		// must terminate the import process gracefully.
		val, err := d.ifdUint(p)
		if err != nil {
			return err
		}
		for _, v := range val {
			if v == 1 {
				// tSampleFormat == 2 for LogLuv/LogL with bpp == 16
				// tSampleFormat == 3 only when bpp == 32
				return UnsupportedError("sample format")
			}
		}
		// default:
		// 	fmt.Println(tag, "-", p)
	}
	return nil
}

// ifdUint decodes the IFD entry in p, which must be of the Byte, Short
// or Long type, and returns the decoded uint values.
func (d *idf) ifdUint(p []byte) (u []uint, err error) {
	var raw []byte
	datatype := d.byteOrder.Uint16(p[2:4])
	count := d.byteOrder.Uint32(p[4:8])
	if datalen := lengths[datatype] * count; datalen > 4 {
		// The IFD contains a pointer to the real value.
		raw = make([]byte, datalen)
		_, err = d.r.ReadAt(raw, int64(d.byteOrder.Uint32(p[8:12])))
	} else {
		raw = p[8 : 8+datalen]
	}
	if err != nil {
		return nil, err
	}

	u = make([]uint, count)
	switch datatype {
	case dtByte:
		for i := uint32(0); i < count; i++ {
			u[i] = uint(raw[i])
		}
	case dtShort:
		for i := uint32(0); i < count; i++ {
			u[i] = uint(d.byteOrder.Uint16(raw[2*i : 2*(i+1)]))
		}
	case dtLong:
		for i := uint32(0); i < count; i++ {
			u[i] = uint(d.byteOrder.Uint32(raw[4*i : 4*(i+1)]))
		}
	case dtDouble:
		for i := uint32(0); i < count; i++ {
			u[i] = uint(d.byteOrder.Uint64(raw[8*i : 8*(i+1)]))

			// var v float64
			// binary.Read(bytes.NewBuffer(raw[8*i:8*(i+1)]), d.byteOrder, &v)
			// fmt.Println(v)
		}
	default:
		return nil, UnsupportedError("data type")
	}
	return u, nil
}
