package tiff_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdouchement/hdr"
	_ "github.com/mdouchement/hdr/codec/hli"
	_ "github.com/mdouchement/hdr/codec/rgbe"
	"github.com/mdouchement/hdrtool"
	_ "github.com/mdouchement/tiff"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRGB32(t *testing.T) {
	base, err := load("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.tif.hdr")
	assert.NoError(t, err)

	rgb32, err := load("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.RGB32.tiff")
	assert.NoError(t, err)

	ssim := hdrtool.HDRSSIM(base, rgb32)
	assert.Equal(t, float64(1), ssim)
}

func TestLogluv(t *testing.T) {
	base, err := load("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.tif.hdr")
	assert.NoError(t, err)

	logluv, err := load("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.tif")
	assert.NoError(t, err)

	ssim := hdrtool.HDRSSIM(base, logluv)
	assert.Equal(t, float64(0.9999744835497351), ssim)
}

func TestDNG(t *testing.T) {
	base, err := load("https://github.com/mdouchement/tiff/releases/download/null/DJI_mavic_randomground.hli")
	assert.NoError(t, err)

	dng, err := load("https://github.com/mdouchement/tiff/releases/download/null/DJI_mavic_randomground.dng")
	assert.NoError(t, err)

	ssim := hdrtool.HDRSSIM(base, dng)
	assert.Equal(t, float64(0.9999999999906051), ssim)
}

///////////////////////////
//                       //
// Benchmarks            //
//                       //
///////////////////////////

// go test -run=NONE -bench=.

var hdrimage image.Image

func BenchmarkRGB32(b *testing.B) {
	data, err := read("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.RGB32.tiff")
	assert.NoError(b, err)

	var m image.Image
	for n := 0; n < b.N; n++ {
		m, _, err = image.Decode(bytes.NewBuffer(data))
	}
	assert.NoError(b, err)
	hdrimage = m
}

func BenchmarkLogluv(b *testing.B) {
	data, err := read("https://github.com/mdouchement/tiff/releases/download/null/84y7-StanfordMemorialChurch.tif")
	assert.NoError(b, err)

	var m image.Image
	for n := 0; n < b.N; n++ {
		m, _, err = image.Decode(bytes.NewBuffer(data))
	}
	assert.NoError(b, err)
	hdrimage = m
}

func BenchmarkDNG(b *testing.B) {
	data, err := read("https://github.com/mdouchement/tiff/releases/download/null/DJI_mavic_randomground.dng")
	assert.NoError(b, err)

	var m image.Image
	for n := 0; n < b.N; n++ {
		m, _, err = image.Decode(bytes.NewBuffer(data))
	}
	assert.NoError(b, err)
	hdrimage = m
}

///////////////////////////
//                       //
// Download / Cache      //
//                       //
///////////////////////////

var cachedir = ".tmp"

func load(url string) (hdr.Image, error) {
	dst, err := download(url)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(dst)
	if err != nil {
		return nil, errors.Wrap(err, "could not open image")
	}
	defer f.Close()

	m, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode image")
	}

	hm, ok := m.(hdr.Image)
	if !ok {
		return nil, errors.New("not an HDR image")
	}
	return hm, nil
}

func read(url string) ([]byte, error) {
	dst, err := download(url)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(dst)
	return b, errors.Wrap(err, "could not read data")
}

func download(url string) (string, error) {
	dst, ok := cache(url)
	if ok {
		// fmt.Printf("%s\n  ⇨ retreived from cache %s\n", url, dst)
		return dst, nil
	}

	os.MkdirAll(cachedir, 0755)

	// Initialize HTTP client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 100 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}

	// Download remote file
	resp, err := client.Get(url)
	if err != nil {
		return dst, errors.Wrap(err, "could not download file")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dst, errors.Errorf("bad response status: %s", resp.Status)
	}

	f, err := os.Create(dst)
	if err != nil {
		return dst, errors.Wrap(err, "could not create file")
	}
	defer f.Close()

	// Copy file to disk
	if _, err := io.Copy(f, resp.Body); err != nil {
		return dst, errors.Wrap(err, "could not copy data to file")
	}

	return dst, errors.Wrap(f.Sync(), "could not flush data to disk")
}

func cache(url string) (string, bool) {
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))
	p := filepath.Join(cachedir, sum)
	return p, exists(p)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true // ignoring error
}
