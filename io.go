package gogd

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/KarpelesLab/gowebp"
	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
)

// WebPLossless is the quality sentinel for [ImageWEBP] that selects
// lossless (VP8L) encoding — the same value as PHP's IMG_WEBP_LOSSLESS.
const WebPLossless = 101

// errNilImage is returned by encoders when passed a nil *Image.
var errNilImage = errors.New("gogd: nil image")

// ImageInterlace gets or sets the interlace flag. When enable is
// provided the flag is updated; the return value is 1 if interlace is
// currently on, 0 otherwise. Note: the Go PNG and JPEG encoders in the
// stdlib don't expose an interlace option, so the flag is currently
// stored for API compatibility but not plumbed through to encoders.
func ImageInterlace(img *Image, enable ...bool) int {
	if img == nil {
		return 0
	}
	if len(enable) > 0 {
		img.interlace = enable[0]
	}
	if img.interlace {
		return 1
	}
	return 0
}

// ImagePNG writes img to w as PNG. Accepts any [image.Image].
func ImagePNG(img image.Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return png.Encode(w, img)
}

// ImageJPEG writes img to w as JPEG. quality is 0..100; pass -1 to use the
// default (75). Higher values produce larger, better-looking files.
// Accepts any [image.Image].
func ImageJPEG(img image.Image, w io.Writer, quality int) error {
	if img == nil {
		return errNilImage
	}
	opts := &jpeg.Options{Quality: 75}
	if quality >= 0 {
		if quality > 100 {
			quality = 100
		}
		opts.Quality = quality
	}
	return jpeg.Encode(w, img, opts)
}

// ImageGIF writes img to w as GIF. Truecolor images are quantised to a
// 256-entry palette by the stdlib encoder. Accepts any [image.Image].
func ImageGIF(img image.Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return gif.Encode(w, img, nil)
}

// ImageBMP writes img to w as BMP. Accepts any [image.Image].
func ImageBMP(img image.Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return bmp.Encode(w, img)
}

// ImageWEBP writes img to w as WebP. quality selects the encoding mode:
//
//   - -1: default (lossy at quality 80)
//   - 0..100: lossy VP8 at that quality level (higher = larger, better-looking)
//   - [WebPLossless] (101): lossless VP8L, pixel-perfect
//
// Accepts any [image.Image].
func ImageWEBP(img image.Image, w io.Writer, quality int) error {
	if img == nil {
		return errNilImage
	}
	opts := &gowebp.Options{Method: 4}
	switch {
	case quality < 0:
		opts.Lossy = true
		opts.Quality = 80
	case quality == WebPLossless:
		opts.Lossy = false
	case quality > 100:
		opts.Lossy = true
		opts.Quality = 100
	default:
		opts.Lossy = true
		opts.Quality = float32(quality)
	}
	return gowebp.Encode(w, img, opts)
}

// ImageCreateFromPNG decodes a PNG image from r.
func ImageCreateFromPNG(r io.Reader) (*Image, error) {
	m, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// ImageCreateFromJPEG decodes a JPEG image from r.
func ImageCreateFromJPEG(r io.Reader) (*Image, error) {
	m, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// ImageCreateFromGIF decodes a GIF image from r. Only the first frame is
// returned, matching PHP gd.
func ImageCreateFromGIF(r io.Reader) (*Image, error) {
	m, err := gif.Decode(r)
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// ImageCreateFromBMP decodes a BMP image from r.
func ImageCreateFromBMP(r io.Reader) (*Image, error) {
	m, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// ImageCreateFromWEBP decodes a WebP image from r. WebP encoding is not
// currently implemented by gogd.
func ImageCreateFromWEBP(r io.Reader) (*Image, error) {
	m, err := webp.Decode(r)
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// ImageCreateFromString decodes an image whose format is detected from
// the header bytes. Supports every format gogd can decode: PNG, JPEG,
// GIF, BMP, WebP.
func ImageCreateFromString(data []byte) (*Image, error) {
	m, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return fromStdImage(m), nil
}

// fromStdImage wraps a decoded stdlib image as a *Image, normalising its
// origin to (0, 0).
func fromStdImage(src image.Image) *Image {
	src = normalizeOrigin(src)
	switch m := src.(type) {
	case *image.NRGBA:
		return newImageFromNRGBA(m)
	case *image.Paletted:
		return newImageFromPaletted(m)
	}
	b := src.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), src, b.Min, draw.Src)
	return newImageFromNRGBA(nrgba)
}

func newImageFromNRGBA(m *image.NRGBA) *Image {
	return &Image{
		nrgba:         m,
		alphaBlending: true,
		transparent:   ColorNone,
		thickness:     1,
		interpolation: ImgBilinearFixed,
		resolutionX:   96,
		resolutionY:   96,
	}
}

func newImageFromPaletted(m *image.Paletted) *Image {
	return &Image{
		pal:           m,
		transparent:   ColorNone,
		thickness:     1,
		interpolation: ImgBilinearFixed,
		resolutionX:   96,
		resolutionY:   96,
	}
}

// normalizeOrigin returns src translated so that its bounds start at (0, 0).
// If src already starts at the origin it is returned unchanged.
func normalizeOrigin(src image.Image) image.Image {
	b := src.Bounds()
	if b.Min.X == 0 && b.Min.Y == 0 {
		return src
	}
	switch s := src.(type) {
	case *image.NRGBA:
		out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(out, out.Bounds(), s, b.Min, draw.Src)
		return out
	case *image.Paletted:
		out := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), s.Palette)
		draw.Draw(out, out.Bounds(), s, b.Min, draw.Src)
		return out
	}
	return src
}
