# TIFF codec for HDR images

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/mdouchement/tiff)

A Golang TIFF codec for HDRi formats. This package is meant to be used with [mdouchement/hdr](https://github.com/mdouchement/hdr).

- Only decoder is implemented.

## Photometric Interpretation

- RGB - 32 bit floating point
- LogL - Luminance GrayScale (LogLuv without u & v parts)
- LogLuv - True colors (32 bits only. No support of 24 bits at the moment)

## Compression

- None (Uncompressed)
- LZW
- Deflate (old and new)
- PackBits
- SGI Log RLE

## License

**BSD-style**

This package carries the same license as Golang's [image/tiff](https://github.com/golang/image/tree/master/tiff) package. Because all this package's skeleton and some piece of code come from the `image/tiff` package.


## Contributing

All PRs are welcome.

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request

As possible, run the following commands to format and lint the code:

```sh
# Format
find . -name '*.go' -not -path './vendor*' -exec gofmt -s -w {} \;

# Lint
gometalinter --config=gometalinter.json ./...
```
