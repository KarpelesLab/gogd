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

	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
)

// errNilImage is returned by encoders when passed a nil *Image.
var errNilImage = errors.New("gogd: nil image")

// ImagePNG writes img to w as PNG.
func ImagePNG(img *Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return png.Encode(w, img)
}

// ImageJPEG writes img to w as JPEG. quality is 0..100; pass -1 to use the
// default (75). Higher values produce larger, better-looking files.
func ImageJPEG(img *Image, w io.Writer, quality int) error {
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
// 256-entry palette by the stdlib encoder.
func ImageGIF(img *Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return gif.Encode(w, img, nil)
}

// ImageBMP writes img to w as BMP.
func ImageBMP(img *Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	return bmp.Encode(w, img)
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
