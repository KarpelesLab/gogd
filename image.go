package gogd

import (
	"image"
	"image/color"
)

const (
	// AlphaOpaque is the gd alpha value for a fully opaque pixel.
	AlphaOpaque = 0
	// AlphaTransparent is the gd alpha value for a fully transparent pixel.
	AlphaTransparent = 127
	// AlphaMax is the largest valid gd alpha value.
	AlphaMax = 127
)

// Color is a gd color. In truecolor mode the int is packed as
// (alpha<<24)|(red<<16)|(green<<8)|blue where alpha is in the gd range
// 0..127 (0 = opaque, 127 = fully transparent). In palette mode it is the
// index into the image's color palette.
type Color int

// ColorNone represents an absent or undefined color. PHP returns -1.
const ColorNone Color = -1

// Image is a gd image. Use [ImageCreateTrueColor] or [ImageCreate] to make
// one. Image implements the [image.Image] interface, so it can be handed
// directly to functions in image, image/draw and similar packages.
type Image struct {
	nrgba *image.NRGBA    // non-nil iff truecolor
	pal   *image.Paletted // non-nil iff palette

	alphaBlending bool
	saveAlpha     bool
	transparent   Color
	thickness     int
}

// ImageCreateTrueColor returns a new truecolor Image of the given size.
// The pixel buffer is filled with opaque black, matching PHP gd.
func ImageCreateTrueColor(width, height int) *Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	m := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 3; i < len(m.Pix); i += 4 {
		m.Pix[i] = 0xff
	}
	return &Image{
		nrgba:         m,
		alphaBlending: true,
		transparent:   ColorNone,
		thickness:     1,
	}
}

// ImageCreate returns a new palette Image of the given size. The palette
// starts empty; the first color allocated via [ImageColorAllocate] becomes
// the background fill (every pixel defaults to palette index 0).
func ImageCreate(width, height int) *Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	m := image.NewPaletted(image.Rect(0, 0, width, height), color.Palette{})
	return &Image{
		pal:         m,
		transparent: ColorNone,
		thickness:   1,
	}
}

// ImageDestroy releases the image. In Go memory is garbage-collected, so
// this is effectively a no-op kept for API parity with PHP gd.
func ImageDestroy(img *Image) bool {
	if img == nil {
		return false
	}
	img.nrgba = nil
	img.pal = nil
	return true
}

// ImageIsTrueColor reports whether img is a truecolor image.
func ImageIsTrueColor(img *Image) bool {
	return img != nil && img.nrgba != nil
}

// ImageSX returns the width of img, or 0 if img is nil.
func ImageSX(img *Image) int {
	if img == nil {
		return 0
	}
	return img.Bounds().Dx()
}

// ImageSY returns the height of img, or 0 if img is nil.
func ImageSY(img *Image) int {
	if img == nil {
		return 0
	}
	return img.Bounds().Dy()
}

// Width is a Go-idiomatic alias for [ImageSX].
func (img *Image) Width() int { return ImageSX(img) }

// Height is a Go-idiomatic alias for [ImageSY].
func (img *Image) Height() int { return ImageSY(img) }

// IsTrueColor is a Go-idiomatic alias for [ImageIsTrueColor].
func (img *Image) IsTrueColor() bool { return ImageIsTrueColor(img) }

// Bounds implements [image.Image].
func (img *Image) Bounds() image.Rectangle {
	if img == nil {
		return image.Rectangle{}
	}
	if img.nrgba != nil {
		return img.nrgba.Bounds()
	}
	if img.pal != nil {
		return img.pal.Bounds()
	}
	return image.Rectangle{}
}

// ColorModel implements [image.Image].
func (img *Image) ColorModel() color.Model {
	if img != nil && img.pal != nil {
		return img.pal.ColorModel()
	}
	return color.NRGBAModel
}

// At implements [image.Image].
func (img *Image) At(x, y int) color.Color {
	if img == nil {
		return color.Transparent
	}
	if img.nrgba != nil {
		return img.nrgba.At(x, y)
	}
	if img.pal != nil {
		return img.pal.At(x, y)
	}
	return color.Transparent
}

// compile-time check that *Image satisfies image.Image.
var _ image.Image = (*Image)(nil)
