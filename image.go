package gogd

import (
	"image"
	"image/color"
	"image/draw"
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
	nrgba   *image.NRGBA    // non-nil iff truecolor
	pal     *image.Paletted // non-nil iff palette
	generic draw.Image      // non-nil when wrapping an arbitrary stdlib image

	alphaBlending bool
	saveAlpha     bool
	antialias     bool
	interlace     bool
	transparent   Color
	thickness     int
	interpolation int
	resolutionX   int
	resolutionY   int
	clip          image.Rectangle // zero => whole image
	style         []Color         // set via ImageSetStyle; consulted for ColorStyled
	brush         *Image          // set via ImageSetBrush; painted per line pixel for ColorBrushed
	tile          *Image          // set via ImageSetTile; used for ColorTiled fills
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
		interpolation: ImgBilinearFixed,
		resolutionX:   96,
		resolutionY:   96,
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
		pal:           m,
		transparent:   ColorNone,
		thickness:     1,
		interpolation: ImgBilinearFixed,
		resolutionX:   96,
		resolutionY:   96,
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

// ImageIsTrueColor reports whether img is a truecolor image. Accepts
// any [image.Image]; stdlib palette types report false, everything else
// (including gogd's truecolor mode) reports true.
func ImageIsTrueColor(img image.Image) bool {
	if img == nil {
		return false
	}
	if g, ok := img.(*Image); ok {
		return g != nil && g.nrgba != nil
	}
	_, isPal := img.(*image.Paletted)
	return !isPal
}

// ImageSX returns the width of img, or 0 if img is nil.
func ImageSX(img image.Image) int {
	if img == nil {
		return 0
	}
	return img.Bounds().Dx()
}

// ImageSY returns the height of img, or 0 if img is nil.
func ImageSY(img image.Image) int {
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
	if img.generic != nil {
		return img.generic.Bounds()
	}
	return image.Rectangle{}
}

// ColorModel implements [image.Image].
func (img *Image) ColorModel() color.Model {
	if img != nil {
		if img.pal != nil {
			return img.pal.ColorModel()
		}
		if img.generic != nil {
			return img.generic.ColorModel()
		}
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
	if img.generic != nil {
		return img.generic.At(x, y)
	}
	return color.Transparent
}

// Set implements [draw.Image]. Setting a pixel goes through the
// underlying truecolor or palette buffer without honouring gd state
// (alpha blending, clipping, transparent index); callers that want
// those semantics should use [ImageSetPixel].
func (img *Image) Set(x, y int, c color.Color) {
	if img == nil {
		return
	}
	if img.nrgba != nil {
		img.nrgba.Set(x, y, c)
		return
	}
	if img.pal != nil {
		img.pal.Set(x, y, c)
		return
	}
	if img.generic != nil {
		img.generic.Set(x, y, c)
	}
}

// asImage returns img as a *Image. If img is already a *Image it is
// returned unchanged (state preserved). *image.NRGBA and *image.Paletted
// are wrapped with default gd state; any other [draw.Image] is wrapped
// via a generic path that uses the interface's Set/At methods. Read-only
// images that aren't a draw.Image (e.g. *image.YCbCr from a jpeg
// decoder) return nil — callers should convert first.
func asImage(img image.Image) *Image {
	if img == nil {
		return nil
	}
	if g, ok := img.(*Image); ok {
		return g
	}
	switch m := img.(type) {
	case *image.NRGBA:
		return newImageFromNRGBA(m)
	case *image.Paletted:
		return newImageFromPaletted(m)
	}
	if d, ok := img.(draw.Image); ok {
		return &Image{
			generic:       d,
			alphaBlending: true,
			transparent:   ColorNone,
			thickness:     1,
			interpolation: ImgBilinearFixed,
			resolutionX:   96,
			resolutionY:   96,
		}
	}
	return nil
}

// compile-time check that *Image satisfies image.Image and draw.Image.
var (
	_ image.Image = (*Image)(nil)
)
