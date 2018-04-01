package tiff

import (
	"fmt"
	"math"
)

// A FormatError reports that the input is not a valid TIFF image.
type FormatError string

func (e FormatError) Error() string {
	return fmt.Sprintf("tiff: invalid format: %s", string(e))
}

// An UnsupportedError reports that the input uses a valid but
// unimplemented feature.
type UnsupportedError string

func (e UnsupportedError) Error() string {
	return fmt.Sprintf("tiff: unsupported feature: %s", string(e))
}

// An InternalError reports that an internal error was encountered.
type InternalError string

func (e InternalError) Error() string {
	return fmt.Sprintf("tiff: internal error: %s", string(e))
}

// minInt returns the smaller of x or y.
func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func tagname(t uint16) string {
	switch t {
	case tBitsPerSample:
		return "BitsPerSample"
	case tExtraSamples:
		return "BitsPerSample"
	case tPhotometricInterpretation:
		return "PhotometricInterpretation"
	case tCompression:
		return "Compression"
	case tPredictor:
		return "Predictor"
	case tStripOffsets:
		return "StripOffsets"
	case tStripByteCounts:
		return "StripByteCounts"
	case tSamplesPerPixel:
		return "SamplesPerPixel"
	case tRowsPerStrip:
		return "RowsPerStrip"
	case tTileWidth:
		return "TileWidth"
	case tTileLength:
		return "TileLength"
	case tTileOffsets:
		return "TileOffsets"
	case tTileByteCounts:
		return "TileByteCounts"
	case tPlanarConfiguration:
		return "PlanarConfiguration"
	case tImageLength:
		return "ImageLength"
	case tImageWidth:
		return "ImageWidth"
	case tStonits:
		return "StoNits"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

func valuename(tag uint16, value []uint) string {
	var v interface{}
	switch tag {
	case tPhotometricInterpretation:
		switch value[0] {
		case pWhiteIsZero:
			v = "WhiteIsZero"
		case pBlackIsZero:
			v = "BlackIsZero"
		case pRGB:
			v = "RGB"
		case pPaletted:
			v = "Paletted"
		case pTransMask:
			v = "TransMask"
		case pCMYK:
			v = "CMYK"
		case pYCbCr:
			v = "YCbCr"
		case pCIELab:
			v = "CIE-Lab"
		case pLogL:
			v = "LogL (GrayScale)"
		case pLogLuv:
			v = "SGI LogLuv (Color)"
		default:
			v = value[0]
		}
	case tCompression:
		switch value[0] {
		case cNone:
			v = "None"
		case cCCITT:
			v = "CCITT"
		case cG3:
			v = "Group 3 Fax"
		case cG4:
			v = "Group 4 Fax"
		case cLZW:
			v = "LZW"
		case cJPEGOld:
			v = "Old JPEG"
		case cJPEG:
			v = "JPEG"
		case cDeflate:
			v = "Deflate (zlib compression)"
		case cPackBits:
			v = "PackBits"
		case cDeflateOld:
			v = "Old Deflate"
		case cSGILogRLE:
			v = "SGI Log RLE"
		case cSGILog24Packed:
			v = "SGI Log 24 bits Packed"
		case cLossyJPEG:
			v = "Lossy JPEG"
		default:
			v = value[0]
		}
	case tStripOffsets:
		v = fmt.Sprintf("contains %d offset entries", len(value))
	case tStripByteCounts:
		v = fmt.Sprintf("contains %d byte-count entries", len(value))
	case tSamplesPerPixel:
		fallthrough
	case tRowsPerStrip:
		fallthrough
	case tTileWidth:
		fallthrough
	case tTileLength:
		fallthrough
	case tImageLength:
		fallthrough
	case tImageWidth:
		v = value[0]
	case tPlanarConfiguration:
		switch value[0] {
		case 1:
			v = "Contiguous (aka RGBRGBRGBRGB)"
		case 2:
			v = "Separate (aka RRRRGGGGBBBB)"
		}
	case tStonits:
		v = math.Float64frombits(uint64(value[0]))
	default:
		v = value
	}
	return fmt.Sprintf("%v", v)
}
