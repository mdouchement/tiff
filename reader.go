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
// http://www.barrypearson.co.uk/articles/dng/specification.htm
// https://helpx.adobe.com/photoshop/digital-negative.html
// https://www.adobe.com/content/dam/acom/en/products/photoshop/pdfs/dng_spec_1.4.0.0.pdf
// https://rcsumner.net/raw_guide/RAWguide.pdf (processing workflow)

import (
	"image"
	"io"

	"github.com/mdouchement/hdr"
)

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
		if d.bpp == 16 || d.bpp == 8 {
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
