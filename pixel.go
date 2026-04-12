package gogd

import (
	"image"
	"image/color"
)

// ImageSetPixel sets the pixel at (x, y) to c. For truecolor images c is a
// packed gd color (typically returned by [ImageColorAllocate]); if alpha
// blending is enabled (the default) c is composited over the existing
// pixel. For palette images c is a palette index.
//
// Returns false if (x, y) is outside the image bounds or if c is not a
// valid palette index.
func ImageSetPixel(img *Image, x, y int, c Color) bool {
	if img == nil {
		return false
	}
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		return false
	}
	if img.nrgba != nil {
		src := gdColorToNRGBA(c)
		if img.alphaBlending && src.A != 0xff {
			src = blendNRGBA(img.nrgba.NRGBAAt(x, y), src)
		}
		img.nrgba.SetNRGBA(x, y, src)
		return true
	}
	if img.pal == nil || int(c) < 0 || int(c) >= len(img.pal.Palette) {
		return false
	}
	img.pal.SetColorIndex(x, y, uint8(c))
	return true
}

// ImageColorAt returns the gd color at (x, y). For truecolor images it is
// the packed RGBA value; for palette images it is the palette index.
// Returns [ColorNone] for a nil image or out-of-bounds coordinates.
func ImageColorAt(img *Image, x, y int) Color {
	if img == nil {
		return ColorNone
	}
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		return ColorNone
	}
	if img.nrgba != nil {
		c := img.nrgba.NRGBAAt(x, y)
		return packGDColor(int(c.R), int(c.G), int(c.B), stdAlphaToGD(c.A))
	}
	if img.pal == nil {
		return ColorNone
	}
	return Color(img.pal.ColorIndexAt(x, y))
}

// ImageAlphaBlending toggles alpha-blended drawing on a truecolor image.
// Returns the previous value. The flag has no effect on palette images.
func ImageAlphaBlending(img *Image, enable bool) bool {
	if img == nil {
		return false
	}
	prev := img.alphaBlending
	img.alphaBlending = enable
	return prev
}

// ImageSaveAlpha controls whether to preserve the full alpha channel when
// the image is encoded. Returns the previous value.
func ImageSaveAlpha(img *Image, enable bool) bool {
	if img == nil {
		return false
	}
	prev := img.saveAlpha
	img.saveAlpha = enable
	return prev
}

// gdColorToNRGBA converts a packed gd color into a color.NRGBA.
func gdColorToNRGBA(c Color) color.NRGBA {
	r, g, b, a := unpackGDColor(c)
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: gdAlphaToStdAlpha(a)}
}

// blendNRGBA composites src over dst in non-premultiplied RGBA space.
func blendNRGBA(dst, src color.NRGBA) color.NRGBA {
	if src.A == 0 {
		return dst
	}
	if src.A == 0xff || dst.A == 0 {
		return src
	}
	sa := uint32(src.A)
	da := uint32(dst.A)
	outA := sa + da*(0xff-sa)/0xff
	if outA == 0 {
		return color.NRGBA{}
	}
	mix := func(s, d uint8) uint8 {
		return uint8((uint32(s)*sa + uint32(d)*da*(0xff-sa)/0xff) / outA)
	}
	return color.NRGBA{
		R: mix(src.R, dst.R),
		G: mix(src.G, dst.G),
		B: mix(src.B, dst.B),
		A: uint8(outA),
	}
}
