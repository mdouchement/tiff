package tiff

import (
	"fmt"
	"image"
	"math"

	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/hdrcolor"
	"github.com/mdouchement/tiff/bayer"
)

func (d *decoder) decodeColorFilterArray(dst image.Image, xmin, ymin, xmax, ymax int) error {
	// Apply horizontal predictor if necessary.
	// In this case, p contains the color difference to the preceding pixel.
	// See page 64-65 of the spec.
	if d.firstVal(tPredictor) > prNone {
		return UnsupportedError("predictor")
	}

	rMaxX := minInt(xmax, dst.Bounds().Max.X)
	rMaxY := minInt(ymax, dst.Bounds().Max.Y)

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
		fmt.Println("You may need to linearize the CFA:", t.val)
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
	if t, exists := d.features[tAsShotNeutral]; exists {
		// Invert the values and then rescale them all so that the green multiplier is 1.
		opts.WhiteBalance = make([]float64, len(t.val))
		for i := range t.val {
			opts.WhiteBalance[i] = 1 / t.asFloat(i)
		}
		opts.WhiteBalance[0] /= opts.WhiteBalance[1]
		opts.WhiteBalance[1] /= opts.WhiteBalance[1]
		opts.WhiteBalance[2] /= opts.WhiteBalance[1]
	} else {
		opts.WhiteBalance = []float64{1, 1, 1}
	}

	// Step 3 - Demosaicing
	bayer := bayer.NewBilinear(d.buf, opts)

	// Step 4 - Color Space Correction
	// camToXYZ := []float64{}
	// if t, exists := d.features[tColorMatrix2]; exists {
	// 	data := make([]float64, len(t.val))
	// 	for i := range t.val {
	// 		data[i] = t.asFloat(i)
	// 	}
	// 	xyzToCam := mat.NewDense(3, 3, data) // nbOfRows should be equal to len(d.features[tCFAPlaneColor].val)
	// 	var im mat.Dense
	// 	im.Inverse(xyzToCam)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(0)...)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(1)...)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(2)...)
	// } else if t, exists := d.features[tColorMatrix1]; exists {
	// 	data := make([]float64, len(t.val))
	// 	for i := range t.val {
	// 		data[i] = t.asFloat(i)
	// 	}
	// 	xyzToCam := mat.NewDense(3, 3, data) // nbOfRows should be equal to len(d.features[tCFAPlaneColor].val)
	// 	var im mat.Dense
	// 	im.Inverse(xyzToCam)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(0)...)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(1)...)
	// 	camToXYZ = append(camToXYZ, im.RawRowView(2)...)
	// } else {
	// 	// sRBG->XYZ (D65)
	// 	camToXYZ = []float64{
	// 		0.4124564, 0.3575761, 0.1804375,
	// 		0.2126729, 0.7151522, 0.0721750,
	// 		0.0193339, 0.1191920, 0.9503041,
	// 	}
	// }
	camToXYZ := []float64{
		0.4124564, 0.3575761, 0.1804375,
		0.2126729, 0.7151522, 0.0721750,
		0.0193339, 0.1191920, 0.9503041,
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

	return nil
}
