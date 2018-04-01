package tiff

// Resources:
// https://github.com/golang/image/tree/master/tiff
// https://www.fileformat.info/format/tiff/egff.htm
// http://www.awaresystems.be/imaging/tiff.html
// http://www.anyhere.com/gward/pixformat/tiffluv.html (LogL / LogLuv)
// http://www.anyhere.com/gward/papers/jgtpap1.pdf (LogLuv spec paper)

import (
	"image"
	"io"

	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/format"
	"github.com/mdouchement/hdr/hdrcolor"
)

// decode decodes the raw data of an image.
// It reads from d.buf and writes the strip or tile into dst.
func (d *decoder) decode(dst image.Image, xmin, ymin, xmax, ymax int) error {
	rMaxX := minInt(xmax, dst.Bounds().Max.X)
	rMaxY := minInt(ymax, dst.Bounds().Max.Y)
	var offset uint
	stonits := d.firstDouble(tStonits)
	if stonits == 0 {
		stonits = 1
	}
	// stonits := math.Pow10(-2)

	// Apply horizontal predictor if necessary.
	// In this case, p contains the color difference to the preceding pixel.
	// See page 64-65 of the spec.
	if d.firstVal(tPredictor) > prNone {
		return UnsupportedError("predictor")
	}

	switch d.mode {
	case mRGB:
		m := dst.(*hdr.RGB)
		for y := ymin; y < rMaxY; y++ {
			for x := xmin; x < rMaxX; x++ {
				R, G, B := format.FromBytes(d.byteOrder, d.buf[offset:offset+12])
				m.SetRGB(x, y, hdrcolor.RGB{R: R, G: G, B: B})
				offset += 12 // RGB is hold on 12 Bytes (4 Bytes per channel)
			}
		}
	case mLogL:
		m := dst.(*hdr.XYZ)
		for y := ymin; y < rMaxY; y++ {
			for x := xmin; x < rMaxX; x++ {
				SLe := format.BytesToUint16(d.buf[offset], d.buf[offset+1])
				Y := format.SLeToY(SLe)
				m.SetXYZ(x, y, hdrcolor.XYZ{X: Y * stonits, Y: Y * stonits, Z: Y * stonits})
				offset += 2 // LogL is hold on 2 bytes (the luminance used in GrayScale)
			}
		}
	case mLogLuv:
		m := dst.(*hdr.XYZ)
		for y := ymin; y < rMaxY; y++ {
			for x := xmin; x < rMaxX; x++ {
				X, Y, Z := format.LogLuvToXYZ(d.buf[offset], d.buf[offset+1], d.buf[offset+2], d.buf[offset+3])
				m.SetXYZ(x, y, hdrcolor.XYZ{X: X * stonits, Y: Y * stonits, Z: Z * stonits})
				offset += 4 // LogLuv is hold on 4 bytes
			}
		}
	}
	return nil
}

//------------------------//
// Reader                 //
//------------------------//

// DecodeConfig returns the color model and dimensions of a TIFF image without
// decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	d, err := newDecoder(r)
	if err != nil {
		return image.Config{}, err
	}
	return d.config, nil
}

// Decode reads a DNG image from r and returns an image.Image.
func Decode(r io.Reader) (m image.Image, err error) {
	d, err := newDecoder(r)
	if err != nil {
		return
	}

	// fmt.Println("=================")
	// fmt.Println(d.String())
	// fmt.Println("=================")

	// ==============================================================
	blockPadding := false
	blockWidth := d.config.Width
	blockHeight := d.config.Height
	blocksAcross := 1
	blocksDown := 1

	if d.config.Width == 0 {
		blocksAcross = 0
	}
	if d.config.Height == 0 {
		blocksDown = 0
	}

	var blockOffsets, blockCounts []uint

	if int(d.firstVal(tTileWidth)) != 0 {
		blockPadding = true

		blockWidth = int(d.firstVal(tTileWidth))
		blockHeight = int(d.firstVal(tTileLength))

		if blockWidth != 0 {
			blocksAcross = (d.config.Width + blockWidth - 1) / blockWidth
		}
		if blockHeight != 0 {
			blocksDown = (d.config.Height + blockHeight - 1) / blockHeight
		}

		blockCounts = d.features[tTileByteCounts]
		blockOffsets = d.features[tTileOffsets]

	} else {
		if int(d.firstVal(tRowsPerStrip)) != 0 {
			blockHeight = int(d.firstVal(tRowsPerStrip))
		}

		if blockHeight != 0 {
			blocksDown = (d.config.Height + blockHeight - 1) / blockHeight
		}

		blockOffsets = d.features[tStripOffsets]
		blockCounts = d.features[tStripByteCounts]
	}

	// Check if we have the right number of strips/tiles, offsets and counts.
	if n := blocksAcross * blocksDown; len(blockOffsets) < n || len(blockCounts) < n {
		return nil, FormatError("inconsistent header")
	}

	// ==============================================================

	bounds := image.Rect(0, 0, d.config.Width, d.config.Height)
	switch d.mode {
	case mRGB:
		if d.bpp == 32 {
			m = hdr.NewRGB(bounds)
		} else {
			err = FormatError("Invalid BitsPerSample for RGB 32 bits floating-point format")
			return
		}
	case mLogL:
		if d.bpp == 16 {
			m = hdr.NewXYZ(bounds)
		} else {
			err = FormatError("Invalid BitsPerSample for LogL format")
			return
		}
	case mLogLuv:
		if d.bpp == 16 {
			m = hdr.NewXYZ(bounds)
		} else {
			err = FormatError("Invalid BitsPerSample for LogLuv format")
			return
		}
	}

	// ==============================================================

	for i := 0; i < blocksAcross; i++ {
		blkW := blockWidth
		if !blockPadding && i == blocksAcross-1 && d.config.Width%blockWidth != 0 {
			blkW = d.config.Width % blockWidth
		}
		for j := 0; j < blocksDown; j++ {
			blkH := blockHeight
			if !blockPadding && j == blocksDown-1 && d.config.Height%blockHeight != 0 {
				blkH = d.config.Height % blockHeight
			}
			offset := int64(blockOffsets[j*blocksAcross+i])
			n := int64(blockCounts[j*blocksAcross+i])

			if err = d.decompress(offset, n, blkW, blkH); err != nil {
				return nil, err
			}

			xmin := i * blockWidth
			ymin := j * blockHeight
			xmax := xmin + blkW
			ymax := ymin + blkH
			err = d.decode(m, xmin, ymin, xmax, ymax)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}

func init() {
	image.RegisterFormat("tiff", leHeader, Decode, DecodeConfig)
	image.RegisterFormat("tiff", beHeader, Decode, DecodeConfig)
}
