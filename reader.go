package tiff

// Resources:
// https://github.com/golang/image/tree/master/tiff
// https://www.fileformat.info/format/tiff/egff.htm
// http://www.awaresystems.be/imaging/tiff.html
// http://www.anyhere.com/gward/pixformat/tiffluv.html (LogL / LogLuv)
// http://www.anyhere.com/gward/papers/jgtpap1.pdf (LogLuv spec paper)
//
// TIFF/EP:
// https://www.awaresystems.be/imaging/tiff/specification/TIFFPM6.pdf (SubIFD Trees)
// https://www.loc.gov/preservation/digital/formats/fdd/fdd000073.shtml (TIFF/EP)
// https://www.loc.gov/preservation/digital/formats/content/tiff_tags.shtml (Tags description)
// DNG:
// https://helpx.adobe.com/photoshop/digital-negative.html
// https://www.adobe.com/content/dam/acom/en/products/photoshop/pdfs/dng_spec_1.4.0.0.pdf
// https://rcsumner.net/raw_guide/RAWguide.pdf (processing workflow)

import (
	"fmt"
	"image"
	"io"
	"math"

	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/format"
	"github.com/mdouchement/hdr/hdrcolor"
	"github.com/mdouchement/tiff/bayer"
	"gonum.org/v1/gonum/mat"
)

// decode decodes the raw data of an image.
// It reads from d.buf and writes the strip or tile into dst.
func (d *decoder) decode(dst image.Image, xmin, ymin, xmax, ymax int) error {
	rMaxX := minInt(xmax, dst.Bounds().Max.X)
	rMaxY := minInt(ymax, dst.Bounds().Max.Y)
	var offset uint
	stonits := d.features[tStonits].double(0)
	if stonits == 0 {
		stonits = 1
	}

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
	case mColorFilterArray:
		// Described workflow -> https://rcsumner.net/raw_guide/RAWguide.pdf
		p, err := bayer.GetPattern(d.features[tCFAPattern].val)
		if err != nil {
			return err
		}
		opts := &bayer.Options{
			ByteOrder: d.byteOrder,
			Depth:     int(d.bpp),
			Width:     rMaxX,
			Height:    rMaxY,
			Pattern:   p,
		}
		// Step 1 - Linearizing + Luminance ReScale used in Bayer.
		if t, exists := d.features[tLinearizationTable]; exists {
			fmt.Println("You need to linearize the CFA:", t.val)
		}
		if t, exists := d.features[tBlackLevel]; exists {
			opts.BlackLevel = t.asFloat(0)
		}
		if t, exists := d.features[tWhiteLevel]; exists {
			opts.WhiteLevel = t.asFloat(0)
		} else {
			opts.WhiteLevel = math.Exp2(float64(d.bpp)) - 1 // Max color channel value
		}

		// Step 2 - White Balancing
		// if t, exists := d.features[tAsShotNeutral]; exists {
		// 	// Invert the values and then rescale them all so that the green multiplier is 1.
		// 	opts.WhiteBalance = make([]float64, len(t.val))
		// 	for i := range t.val {
		// 		opts.WhiteBalance[i] = 1 / t.asFloat(i)
		// 	}
		// 	opts.WhiteBalance[0] /= opts.WhiteBalance[1]
		// 	opts.WhiteBalance[1] /= opts.WhiteBalance[1]
		// 	opts.WhiteBalance[2] /= opts.WhiteBalance[1]
		// } else {
		// 	opts.WhiteBalance = []float64{1, 1, 1}
		// }
		opts.WhiteBalance = []float64{1, 1, 1}

		// Step 3 - Demosaicing
		bayer := bayer.NewBilinear(d.buf, opts)

		// Step 4 - Color Space Correction  TODO
		camToXYZ := []float64{}
		if t, exists := d.features[tColorMatrix2]; exists {
			data := make([]float64, len(t.val))
			for i := range t.val {
				data[i] = t.asFloat(i)
			}
			xyzToCam := mat.NewDense(3, 3, data) // nbOfRows should be equal to len(d.features[tCFAPlaneColor].val)
			var im mat.Dense
			im.Inverse(xyzToCam)
			camToXYZ = append(camToXYZ, im.RawRowView(0)...)
			camToXYZ = append(camToXYZ, im.RawRowView(1)...)
			camToXYZ = append(camToXYZ, im.RawRowView(2)...)
		} else if t, exists := d.features[tColorMatrix1]; exists {
			data := make([]float64, len(t.val))
			for i := range t.val {
				data[i] = t.asFloat(i)
			}
			xyzToCam := mat.NewDense(3, 3, data) // nbOfRows should be equal to len(d.features[tCFAPlaneColor].val)
			var im mat.Dense
			im.Inverse(xyzToCam)
			camToXYZ = append(camToXYZ, im.RawRowView(0)...)
			camToXYZ = append(camToXYZ, im.RawRowView(1)...)
			camToXYZ = append(camToXYZ, im.RawRowView(2)...)
		} else {
			// sRBG->XYZ (D65)
			camToXYZ = []float64{
				0.4124564, 0.3575761, 0.1804375,
				0.2126729, 0.7151522, 0.0721750,
				0.0193339, 0.1191920, 0.9503041,
			}
		}

		// Step 5 - Brightness & Gamma correction TODO (or not because TMO handle it well)

		//
		m := dst.(*hdr.XYZ)
		var r, g, b, X, Y, Z float64
		for y := ymin; y < rMaxY; y++ {
			for x := xmin; x < rMaxX; x++ {
				r, g, b = bayer.At(x, y)

				X = r*camToXYZ[0] + g*camToXYZ[1] + b*camToXYZ[2]
				Y = r*camToXYZ[3] + g*camToXYZ[4] + b*camToXYZ[5]
				Z = r*camToXYZ[6] + g*camToXYZ[7] + b*camToXYZ[8]

				m.SetXYZ(x, y, hdrcolor.XYZ{X: X, Y: Y, Z: Z})
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

		blockCounts = d.features[tTileByteCounts].val
		blockOffsets = d.features[tTileOffsets].val

	} else {
		if int(d.firstVal(tRowsPerStrip)) != 0 {
			blockHeight = int(d.firstVal(tRowsPerStrip))
		}

		if blockHeight != 0 {
			blocksDown = (d.config.Height + blockHeight - 1) / blockHeight
		}

		blockOffsets = d.features[tStripOffsets].val
		blockCounts = d.features[tStripByteCounts].val
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
	case mColorFilterArray:
		if d.bpp == 16 {
			m = hdr.NewXYZ(bounds)
		} else {
			err = FormatError("Invalid BitsPerSample for ColorFilterArray format")
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
