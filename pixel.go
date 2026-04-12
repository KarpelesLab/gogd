package gogd

import (
	"image"
	"image/color"
	"image/draw"
)

// ImageSetPixel sets the pixel at (x, y) to c. For truecolor images c is a
// packed gd color (typically returned by [ImageColorAllocate]); if alpha
// blending is enabled (the default) c is composited over the existing
// pixel. For palette images c is a palette index.
//
// Accepts any [draw.Image]: *gogd.Image uses its state (alpha blending,
// clipping semantics come via the drawing primitives); other stdlib
// images are treated as unstyled truecolor unless they are
// *image.Paletted, in which case c is a palette index.
//
// Returns false if (x, y) is outside the image bounds or c is not a
// valid palette index.
func ImageSetPixel(img draw.Image, x, y int, c Color) bool {
	if img == nil {
		return false
	}
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		return false
	}
	g := asImage(img)
	if g == nil {
		img.Set(x, y, gdColorToNRGBA(c))
		return true
	}
	if g.nrgba != nil {
		src := gdColorToNRGBA(c)
		if g.alphaBlending && src.A != 0xff {
			src = blendNRGBA(g.nrgba.NRGBAAt(x, y), src)
		}
		g.nrgba.SetNRGBA(x, y, src)
		return true
	}
	if g.pal != nil {
		if int(c) < 0 || int(c) >= len(g.pal.Palette) {
			return false
		}
		g.pal.SetColorIndex(x, y, uint8(c))
		return true
	}
	if g.generic != nil {
		src := gdColorToNRGBA(c)
		if g.alphaBlending && src.A != 0xff {
			dst := color.NRGBAModel.Convert(g.generic.At(x, y)).(color.NRGBA)
			src = blendNRGBA(dst, src)
		}
		g.generic.Set(x, y, src)
		return true
	}
	return false
}

// ImageColorAt returns the gd color at (x, y). For truecolor images it is
// the packed RGBA value; for palette images it is the palette index.
// Returns [ColorNone] for a nil image or out-of-bounds coordinates.
// Accepts any [image.Image].
func ImageColorAt(img image.Image, x, y int) Color {
	if img == nil {
		return ColorNone
	}
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		return ColorNone
	}
	if g, ok := img.(*Image); ok {
		if g == nil {
			return ColorNone
		}
		if g.nrgba != nil {
			c := g.nrgba.NRGBAAt(x, y)
			return packGDColor(int(c.R), int(c.G), int(c.B), stdAlphaToGD(c.A))
		}
		if g.pal != nil {
			return Color(g.pal.ColorIndexAt(x, y))
		}
		return ColorNone
	}
	if p, ok := img.(*image.Paletted); ok {
		return Color(p.ColorIndexAt(x, y))
	}
	nc := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	return packGDColor(int(nc.R), int(nc.G), int(nc.B), stdAlphaToGD(nc.A))
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
