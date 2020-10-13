package tiff

import (
	"image"

	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/format"
	"github.com/mdouchement/hdr/hdrcolor"
)

func (d *decoder) decodeLogL(dst image.Image, xmin, ymin, xmax, ymax int) error {
	// Apply horizontal predictor if necessary.
	// In this case, p contains the color difference to the preceding pixel.
	// See page 64-65 of the spec.
	if d.firstVal(tPredictor) > prNone {
		return UnsupportedError("predictor")
	}

	rMaxX := minInt(xmax, dst.Bounds().Max.X)
	rMaxY := minInt(ymax, dst.Bounds().Max.Y)
	var offset uint

	stonits := d.features[tStonits].double(0)
	if stonits == 0 {
		stonits = 1
	}

	m := dst.(*hdr.XYZ)
	for y := ymin; y < rMaxY; y++ {
		for x := xmin; x < rMaxX; x++ {
			SLe := format.BytesToUint16(d.buf[offset], d.buf[offset+1])
			Y := format.SLeToY(SLe)
			m.SetXYZ(x, y, hdrcolor.XYZ{X: Y * stonits, Y: Y * stonits, Z: Y * stonits})
			offset += 2 // LogL is hold on 2 bytes (the luminance used in GrayScale)
		}
	}

	return nil
}
