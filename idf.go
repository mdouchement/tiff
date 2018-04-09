package tiff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

//------------------------//
// Header parser          //
//------------------------//

type (
	idf struct {
		r         io.ReaderAt
		byteOrder binary.ByteOrder
		format    int
		features  map[uint16]tag
		tree      []map[uint16]tag // IDF-Tree
	}
)

func newIDF(r io.ReaderAt) (d *idf, err error) {
	d = &idf{
		r:        r,
		format:   fTIFF,
		features: make(map[uint16]tag),
		tree:     make([]map[uint16]tag, 0),
	}

	p := make([]byte, 8)
	if _, err = d.r.ReadAt(p, 0); err != nil {
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
	if err = d.appendAndParseIDF(0, ifdOffset); err != nil { // Main IDF is at index 0.
		return nil, err
	}

	// Add main IDF data in features map.
	for k, v := range d.tree[0] {
		d.features[k] = v
	}

	// Update file format
	if _, ok := d.features[tDNGVersion]; ok {
		d.format = fDNG
	}

	if subIDFs, ok := d.features[tSubIFDs]; ok {
		// Parse all SubIFD
		for fi, offset := range subIDFs.val {
			if err = d.appendAndParseIDF(fi+1, int64(offset)); err != nil {
				return nil, err
			}
		}
	}

	if d.format == fDNG {
		// Find `Primary image`, the highest-resolution and quality IFD.
		for _, features := range d.tree {
			feature, ok := features[tNewSubFileType]
			if ok && feature.val[0] == sftPrimaryImage {
				// Add/overwrite features with the primary image matadata.
				for k, v := range features {
					d.features[k] = v
				}
				break
			}
		}
	}

	return
}

// firstVal is a convenient accessor of tag#firstVal().
func (d *idf) firstVal(tag uint16) uint {
	return d.features[tag].firstVal()
}

func (d *idf) appendAndParseIDF(fi int, ifdOffset int64) error {
	d.tree = append(d.tree, make(map[uint16]tag)) // Append to `fi` index
	p := make([]byte, 8)

	// The first two bytes contain the number of entries (12 bytes each).
	if _, err := d.r.ReadAt(p[0:2], ifdOffset); err != nil {
		return err
	}
	numItems := int(d.byteOrder.Uint16(p[0:2]))

	// All IFD entries are read in one chunk.
	p = make([]byte, ifdLen*numItems)
	if _, err := d.r.ReadAt(p, ifdOffset+2); err != nil {
		return err
	}

	for i := 0; i < len(p); i += ifdLen {
		if err := d.parseIFD(fi, p[i:i+ifdLen]); err != nil {
			return err
		}
	}

	return nil
}

// parseIFD decides whether the the IFD entry in p is "interesting" and
// stows away the data in the decoder.
func (d *idf) parseIFD(fi int, p []byte) error {
	tid := d.byteOrder.Uint16(p[0:2]) // TagID
	switch tid {
	case tBitsPerSample,
		tExtraSamples,
		tPhotometricInterpretation,
		tCompression,
		tPredictor,
		tNewSubFileType,
		tSubIFDs,
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
		tStonits,
		tCFARepeatPatternDim,
		tCFAPattern,
		tDNGVersion,
		tDNGBackwardVersion,
		tCFAPlaneColor,
		tCFALayout,
		tLinearizationTable,
		tBlackLevel,
		tWhiteLevel,
		tColorMatrix1,
		tColorMatrix2,
		tAsShotNeutral,
		tBaselineExposure:
		val, dt, err := d.ifdUint(p)
		if err != nil {
			return err
		}
		d.tree[fi][tid] = tag{
			id:       tid,
			datatype: dt,
			val:      val,
		}
		// fmt.Println(d.tree[fi][tid])
	case tSampleFormat:
		// Page 27 of the spec: If the SampleFormat is present and
		// the value is not 1 [= unsigned integer data], a Baseline
		// TIFF reader that cannot handle the SampleFormat value
		// must terminate the import process gracefully.
		val, _, err := d.ifdUint(p)
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
// or Long type, and returns the decoded uint values and their datatype.
func (d *idf) ifdUint(p []byte) (u []uint, dt uint, err error) {
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
		return nil, 0, err
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
	case dtRational:
		fallthrough
	case dtSRational:
		fallthrough
	case dtDouble:
		for i := uint32(0); i < count; i++ {
			u[i] = uint(d.byteOrder.Uint64(raw[8*i : 8*(i+1)]))

			// var v float64
			// binary.Read(bytes.NewBuffer(raw[8*i:8*(i+1)]), d.byteOrder, &v)
			// fmt.Println(v)
		}
	default:
		return nil, 0, UnsupportedError("data type")
	}
	return u, uint(datatype), nil
}

func (d *idf) String() string {
	buf := bytes.NewBufferString("")
	switch d.format {
	case fTIFF:
		buf.WriteString("== TIFF ==\n")
	case fDNG:
		buf.WriteString("== DNG ==\n")
	}
	for _, t := range d.features {
		buf.WriteString(fmt.Sprintf("%v\n", t))
	}
	buf.WriteString(fmt.Sprintf("ByteOrder: %v\n", d.byteOrder))
	buf.WriteString(fmt.Sprintf("BPP: %d\n", d.firstVal(tBitsPerSample)))
	buf.WriteString(fmt.Sprintf("Bounds: %dx%d\n", d.firstVal(tImageWidth), d.firstVal(tImageLength)))
	return buf.String()
}
