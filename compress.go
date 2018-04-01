package tiff

import (
	"bufio"
	"io"
)

type byteReader interface {
	io.Reader
	io.ByteReader
}

// unpackBits decodes the PackBits-compressed data in src and returns the
// uncompressed data.
//
// The PackBits compression format is described in section 9 (p. 42)
// of the TIFF spec.
func unpackBits(r io.Reader) ([]byte, error) {
	var n int
	buf := make([]byte, 128)
	dst := make([]byte, 0, 1024)
	br, ok := r.(byteReader)
	if !ok {
		br = bufio.NewReader(r)
	}

	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return dst, nil
			}
			return nil, err
		}
		code := int(int8(b))
		switch {
		case code >= 0:
			n, err = io.ReadFull(br, buf[:code+1])
			if err != nil {
				return nil, err
			}
			dst = append(dst, buf[:n]...)
		case code == -128:
			// No-op.
		default:
			if b, err = br.ReadByte(); err != nil {
				return nil, err
			}
			for j := 0; j < 1-code; j++ {
				buf[j] = b
			}
			dst = append(dst, buf[:1-code]...)
		}
	}
}

// unRLE decodes the Run-Length Encoded data in src and returns the
// uncompressed data. For LogLuv, each of four bytestreams is encoded separately per row.
// This compression is used for LogLuv anf LogL (mode: mLogLuv or LogL).
// blockWidth and blockHeight are the dimmension of the Strip or Tiles.
func unRLE(r io.Reader, mode imageMode, blockWidth, blockHeight int) (dst []byte, err error) {
	br, ok := r.(byteReader)
	if !ok {
		br = bufio.NewReader(r)
	}

	bytesPerPixel := 4 // mLogLuv
	if mode == mLogL {
		bytesPerPixel = 2 // Luminance without chromatic u, v parts
	}

	var b byte
	dst = make([]byte, blockWidth*blockHeight*bytesPerPixel)

	var runLength int

	for row := 0; row < blockHeight; row++ { // Each row (aka scanline for RLE)
		rowOffest := row * blockWidth * bytesPerPixel

		for channel := 0; channel < bytesPerPixel; channel++ { // planar/separate to interleaved/contiguous looping
			offset := channel
			nbOfPixels := blockWidth // per scanline

			for nbOfPixels > 0 {
				// Read RLE property
				if b, err = br.ReadByte(); err != nil {
					return // Never reached by an io.EOF
				}

				if (b & 128) != 0 {
					// a run of the same value
					runLength = int(b) + (2 - 128)
					nbOfPixels -= runLength

					if b, err = br.ReadByte(); err != nil {
						return // Never reached by an io.EOF
					}

					for ; runLength > 0; runLength-- {
						dst[rowOffest+offset] = b
						offset += bytesPerPixel
					}
				} else {
					// a non-run, copy data
					runLength = int(b)
					nbOfPixels -= runLength

					for ; runLength > 0; runLength-- {
						if b, err = br.ReadByte(); err != nil {
							return // Never reached by an io.EOF
						}
						dst[rowOffest+offset] = b
						offset += bytesPerPixel
					}
				}
			}
		}
	}

	return
}
