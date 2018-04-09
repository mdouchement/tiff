package tiff

import (
	"fmt"
	"math"
	"math/big"
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
	case tNewSubFileType:
		return "NewSubFileType"
	case tSubIFDs:
		return "SubIFDs"
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
	case tCFARepeatPatternDim:
		return "CFARepeatPatternDim"
	case tCFAPattern:
		return "CFAPattern"
	case tDNGVersion:
		return "DNG Version"
	case tDNGBackwardVersion:
		return "DNG Backward Version"
	case tCFAPlaneColor:
		return "CFAPlaneColor"
	case tCFALayout:
		return "CFALayout"
	case tLinearizationTable:
		return "LinearizationTable"
	case tBlackLevel:
		return "BlackLevel"
	case tWhiteLevel:
		return "WhiteLevel"
	case tColorMatrix1:
		return "ColorMatrix1"
	case tColorMatrix2:
		return "ColorMatrix2"
	case tAsShotNeutral:
		return "AsShotNeutral"
	case tBaselineExposure:
		return "BaselineExposure"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

func valuename(t tag) string {
	var v interface{}
	switch t.id {
	case tNewSubFileType:
		switch t.firstVal() {
		case sftPrimaryImage:
			v = "Primary image"
		case sftThumbnail:
			v = "Thumbnail/Preview image"
		default:
			v = t.firstVal()
		}
	case tPhotometricInterpretation:
		switch t.firstVal() {
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
		case pColorFilterArray:
			v = "Color Filter Array"
		case pLogL:
			v = "LogL (GrayScale)"
		case pLogLuv:
			v = "SGI LogLuv (Color)"
		default:
			v = t.firstVal()
		}
	case tCompression:
		switch t.firstVal() {
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
			v = "SGI Log Luminance RLE"
		case cSGILog24Packed:
			v = "SGI Log 24-bits packed"
		case cLossyJPEG:
			v = "Lossy JPEG"
		default:
			v = t.firstVal()
		}
	case tStripOffsets:
		v = fmt.Sprintf("contains %d offset entries", len(t.val))
	case tStripByteCounts:
		v = fmt.Sprintf("contains %d byte-count entries", len(t.val))
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
		v = t.firstVal()
	case tPlanarConfiguration:
		switch t.firstVal() {
		case 1:
			v = "Contiguous (aka RGBRGBRGBRGB)"
		case 2:
			v = "Separate (aka RRRRGGGGBBBB)"
		}
	case tStonits:
		v = math.Float64frombits(uint64(t.val[0]))
	case tCFARepeatPatternDim:
		v = fmt.Sprintf("%d CFARepeatRows, %d CFARepeatCols", t.val[0], t.val[1])
	case tCFAPattern:
		v = fmt.Sprintf("%v (%s%s%s%s)", t.val, cfaColors[t.val[0]], cfaColors[t.val[1]], cfaColors[t.val[2]], cfaColors[t.val[3]])
	case tDNGVersion:
		fallthrough
	case tDNGBackwardVersion:
		v = fmt.Sprintf("%d.%d.%d.%d", t.val[0], t.val[1], t.val[2], t.val[3])
	case tCFALayout:
		switch t.firstVal() {
		case 1:
			v = "Rectangular (or square) layout"
		case 2:
			v = "Staggered layout A: even columns are offset down by 1/2 row"
		case 3:
			v = "Staggered layout B: even columns are offset up by 1/2 row"
		case 4:
			v = "Staggered layout C: even rows are offset right by 1/2 column"
		case 5:
			v = "Staggered layout D: even rows are offset left by 1/2 column"
		default:
			v = t.firstVal()
		}
	case tCFAPlaneColor:
		v = fmt.Sprintf("%v (%s%s%s)", t.val, cfaColors[t.val[0]], cfaColors[t.val[1]], cfaColors[t.val[2]])
	case tBaselineExposure:
		v = t.sRational(0)
	default:
		v = formatDatatype(t)
	}
	return fmt.Sprintf("%v", v)
}

func formatDatatype(t tag) interface{} {
	switch t.datatype {
	case dtRational:
		sl := make([]*big.Rat, 0, len(t.val))
		for i := range t.val {
			sl = append(sl, t.rational(i))
		}
		return sl
	case dtSRational:
		sl := make([]*big.Rat, 0, len(t.val))
		for i := range t.val {
			sl = append(sl, t.sRational(i))
		}
		return sl
	case dtDouble:
		sl := make([]float64, 0, len(t.val))
		for i := range t.val {
			sl = append(sl, t.double(i))
		}
		return sl
	default:
		return t.val
	}
}
