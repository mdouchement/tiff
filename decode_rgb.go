package tiff

import (
	"image"

	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/format"
	"github.com/mdouchement/hdr/hdrcolor"
)

func (d *decoder) decodeRGB(dst image.Image, xmin, ymin, xmax, ymax int) error {
	// Apply horizontal predictor if necessary.
	// In this case, p contains the color difference to the preceding pixel.
	// See page 64-65 of the spec.
	if d.firstVal(tPredictor) > prNone {
		return UnsupportedError("predictor")
	}

	rMaxX := minInt(xmax, dst.Bounds().Max.X)
	rMaxY := minInt(ymax, dst.Bounds().Max.Y)
	var offset uint

	m := dst.(*hdr.RGB)
	for y := ymin; y < rMaxY; y++ {
		for x := xmin; x < rMaxX; x++ {
			R, G, B := format.FromBytes(d.byteOrder, d.buf[offset:offset+12])
			m.SetRGB(x, y, hdrcolor.RGB{R: R, G: G, B: B})
			offset += 12 // RGB is hold on 12 Bytes (4 Bytes per channel)
		}
	}

	return nil
}
