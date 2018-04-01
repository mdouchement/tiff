package tiff

// A tiff image file contains one or more images. The metadata
// of each image is contained in an Image File Directory (IFD),
// which contains entries of 12 bytes each and is described
// on page 14-16 of the specification. An IFD entry consists of
//
//  - a tag, which describes the signification of the entry,
//  - the data type and length of the entry,
//  - the data itself or a pointer to it if it is more than 4 bytes.
//
// The presence of a length means that each IFD is effectively an array.

const (
	leHeader = "II\x2A\x00" // Header for little-endian files.
	beHeader = "MM\x00\x2A" // Header for big-endian files.

	ifdLen = 12 // Length of an IFD entry in bytes.
)

// Data types (p. 14-16 of the spec).
const (
	dtByte      = 1
	dtASCII     = 2
	dtShort     = 3
	dtLong      = 4
	dtRational  = 5
	dtSByte     = 6
	dtUndefined = 7
	dtSShort    = 8
	dtSLong     = 9
	dtSRational = 10
	dtFloat     = 11
	dtDouble    = 12
)

// The length of one instance of each data type in bytes.
var lengths = [...]uint32{0, 1, 1, 2, 4, 8, 42, 42, 42, 42, 42, 42, 8} // '42' numbers are juste here to set the dtDouble length.

// Tags (see p. 28-41 of the spec).
const (
	tImageWidth                = 256
	tImageLength               = 257
	tBitsPerSample             = 258
	tCompression               = 259
	tPhotometricInterpretation = 262

	tStripOffsets    = 273
	tSamplesPerPixel = 277
	tRowsPerStrip    = 278
	tStripByteCounts = 279

	tTileWidth      = 322
	tTileLength     = 323
	tTileOffsets    = 324
	tTileByteCounts = 325

	tXResolution         = 282
	tYResolution         = 283
	tPlanarConfiguration = 284
	tResolutionUnit      = 296

	tPredictor    = 317
	tColorMap     = 320
	tExtraSamples = 338
	tSampleFormat = 339

	tStonits = 37439
)

// Compression types (defined in various places in the spec and supplements).
const (
	cNone       = 1
	cCCITT      = 2
	cG3         = 3 // Group 3 Fax.
	cG4         = 4 // Group 4 Fax.
	cLZW        = 5
	cJPEGOld    = 6 // Superseded by cJPEG.
	cJPEG       = 7
	cDeflate    = 8 // zlib compression.
	cPackBits   = 32773
	cDeflateOld = 32946 // Superseded by cDeflate.

	cSGILogRLE      = 34676 // Logluv
	cSGILog24Packed = 34677 // Logluv
	cLossyJPEG      = 34892 // Lossy JPEG is allowed for IFDs that use PhotometricInterpretation = 34892 (LinearRaw) and 8-bit integer data.
)

// Photometric interpretation values (see p. 37 of the spec).
const (
	pWhiteIsZero = 0
	pBlackIsZero = 1
	pRGB         = 2
	pPaletted    = 3
	pTransMask   = 4 // transparency mask
	pCMYK        = 5
	pYCbCr       = 6
	pCIELab      = 8

	pLogL   = 32844 // GrayScale - CIE Log2(L)
	pLogLuv = 32845 // Color - CIE Log2(L) (u',v')
)

// Values for the tPredictor tag (page 64-65 of the spec).
const (
	prNone          = 1
	prHorizontal    = 2
	prFloatingPoint = 3 // Floating point horizontal differencing, a third specification supplement from Adobe
)

// Values for the tResolutionUnit tag (page 18).
const (
	resNone    = 1
	resPerInch = 2 // Dots per inch.
	resPerCM   = 3 // Dots per centimeter.
)

// imageMode represents the mode of the image.
type imageMode int

const (
	mBilevel imageMode = iota
	mPaletted
	mGray
	mGrayInvert
	mRGB
	mRGBA
	mNRGBA
	mNYCbCrA
	mLogL
	mLogLuv
)
