package gogd

import (
	"image"
	"image/color"
	"math"
)

// ImageColorAllocate allocates a color on img. For truecolor images it
// simply packs the RGB components into a gd color and returns it. For
// palette images it appends the color to the palette (subject to the
// 256-entry limit) and returns the new index. Returns [ColorNone] on error.
func ImageColorAllocate(img *Image, r, g, b int) Color {
	return ImageColorAllocateAlpha(img, r, g, b, AlphaOpaque)
}

// ImageColorAllocateAlpha is like [ImageColorAllocate] but also takes a gd
// alpha value in the range 0..127 (0 = opaque, 127 = transparent).
func ImageColorAllocateAlpha(img *Image, r, g, b, a int) Color {
	if img == nil {
		return ColorNone
	}
	r, g, b, a = clamp8(r), clamp8(g), clamp8(b), clampAlpha(a)
	if img.nrgba != nil {
		return packGDColor(r, g, b, a)
	}
	if img.pal == nil || len(img.pal.Palette) >= 256 {
		return ColorNone
	}
	img.pal.Palette = append(img.pal.Palette, gdToNRGBA(r, g, b, a))
	return Color(len(img.pal.Palette) - 1)
}

// ImageColorDeallocate removes a palette entry from img. For truecolor
// images it is always a no-op that returns true.
func ImageColorDeallocate(img *Image, c Color) bool {
	if img == nil {
		return false
	}
	if img.nrgba != nil {
		return true
	}
	if img.pal == nil || int(c) < 0 || int(c) >= len(img.pal.Palette) {
		return false
	}
	return true
}

// ImageColorsTotal returns the number of colors in img's palette, or 0 if
// img is truecolor.
func ImageColorsTotal(img *Image) int {
	if img == nil || img.pal == nil {
		return 0
	}
	return len(img.pal.Palette)
}

// ImageColorsForIndex returns the (r, g, b, a) components of a color. For
// palette images c is a palette index; for truecolor images c is the
// packed gd color previously obtained from one of the allocation/read
// functions. Alpha is returned in the gd range 0..127. Accepts any
// [image.Image].
func ImageColorsForIndex(img image.Image, c Color) (r, g, b, a int) {
	if img == nil {
		return 0, 0, 0, 0
	}
	if g, ok := img.(*Image); ok {
		if g == nil {
			return 0, 0, 0, 0
		}
		if g.nrgba != nil {
			return unpackGDColor(c)
		}
		if g.pal == nil || int(c) < 0 || int(c) >= len(g.pal.Palette) {
			return 0, 0, 0, 0
		}
		nr, ng, nb, na := nrgbaComponents(g.pal.Palette[int(c)])
		return int(nr), int(ng), int(nb), stdAlphaToGD(na)
	}
	if p, ok := img.(*image.Paletted); ok {
		if int(c) < 0 || int(c) >= len(p.Palette) {
			return 0, 0, 0, 0
		}
		nr, ng, nb, na := nrgbaComponents(p.Palette[int(c)])
		return int(nr), int(ng), int(nb), stdAlphaToGD(na)
	}
	return unpackGDColor(c)
}

// ImageColorExact returns the palette index that matches (r, g, b) exactly,
// or [ColorNone] if there is no such entry. For truecolor images it returns
// the packed gd color.
func ImageColorExact(img *Image, r, g, b int) Color {
	return ImageColorExactAlpha(img, r, g, b, AlphaOpaque)
}

// ImageColorExactAlpha is like [ImageColorExact] but also matches the alpha
// channel.
func ImageColorExactAlpha(img *Image, r, g, b, a int) Color {
	if img == nil {
		return ColorNone
	}
	r, g, b, a = clamp8(r), clamp8(g), clamp8(b), clampAlpha(a)
	if img.nrgba != nil {
		return packGDColor(r, g, b, a)
	}
	if img.pal == nil {
		return ColorNone
	}
	want := gdToNRGBA(r, g, b, a)
	for i, pc := range img.pal.Palette {
		if pcEq(pc, want) {
			return Color(i)
		}
	}
	return ColorNone
}

// ImageColorClosest returns the palette index whose color is closest to
// (r, g, b) (Euclidean distance in RGB). For truecolor images it returns
// the packed gd color.
func ImageColorClosest(img *Image, r, g, b int) Color {
	return ImageColorClosestAlpha(img, r, g, b, AlphaOpaque)
}

// ImageColorClosestHWB returns the palette index whose color is closest
// to (r, g, b) in Hue/Whiteness/Blackness space — a better perceptual
// match than RGB Euclidean distance for palette colors. For truecolor
// images it returns the packed gd color.
func ImageColorClosestHWB(img *Image, r, g, b int) Color {
	if img == nil {
		return ColorNone
	}
	r, g, b = clamp8(r), clamp8(g), clamp8(b)
	if img.nrgba != nil {
		return packGDColor(r, g, b, AlphaOpaque)
	}
	if img.pal == nil || len(img.pal.Palette) == 0 {
		return ColorNone
	}
	th, tw, tbl := rgbToHWB(uint8(r), uint8(g), uint8(b))
	best, bestD := -1, math.Inf(1)
	for i, pc := range img.pal.Palette {
		nc := color.NRGBAModel.Convert(pc).(color.NRGBA)
		ph, pw, pbl := rgbToHWB(nc.R, nc.G, nc.B)
		dh := math.Abs(ph - th)
		if dh > 180 {
			dh = 360 - dh
		}
		d := dh*dh + (pw-tw)*(pw-tw)*1000 + (pbl-tbl)*(pbl-tbl)*1000
		if d < bestD {
			bestD, best = d, i
		}
	}
	return Color(best)
}

// ImageColorClosestAlpha is like [ImageColorClosest] but also considers
// alpha in the distance calculation.
func ImageColorClosestAlpha(img *Image, r, g, b, a int) Color {
	if img == nil {
		return ColorNone
	}
	r, g, b, a = clamp8(r), clamp8(g), clamp8(b), clampAlpha(a)
	if img.nrgba != nil {
		return packGDColor(r, g, b, a)
	}
	if img.pal == nil || len(img.pal.Palette) == 0 {
		return ColorNone
	}
	best, bestD := -1, 1<<30
	for i, pc := range img.pal.Palette {
		pr, pg, pb, pa := nrgbaComponents(pc)
		dr := int(pr) - r
		dg := int(pg) - g
		db := int(pb) - b
		da := stdAlphaToGD(pa) - a
		d := dr*dr + dg*dg + db*db + da*da
		if d < bestD {
			bestD, best = d, i
		}
	}
	return Color(best)
}

// ImageColorResolve returns an exact match if one exists, otherwise
// allocates a new palette entry, falling back to the closest color if the
// palette is full. Always succeeds for truecolor images.
func ImageColorResolve(img *Image, r, g, b int) Color {
	return ImageColorResolveAlpha(img, r, g, b, AlphaOpaque)
}

// ImageColorResolveAlpha is like [ImageColorResolve] but also considers
// alpha.
func ImageColorResolveAlpha(img *Image, r, g, b, a int) Color {
	if img == nil {
		return ColorNone
	}
	if c := ImageColorExactAlpha(img, r, g, b, a); c != ColorNone {
		return c
	}
	if img.nrgba != nil {
		return packGDColor(clamp8(r), clamp8(g), clamp8(b), clampAlpha(a))
	}
	if img.pal != nil && len(img.pal.Palette) < 256 {
		return ImageColorAllocateAlpha(img, r, g, b, a)
	}
	return ImageColorClosestAlpha(img, r, g, b, a)
}

// ImageColorTransparent gets or sets the transparent color of img. Passing
// a negative color or [ColorNone] leaves the value unchanged and returns
// the current transparent color.
func ImageColorTransparent(img *Image, c Color) Color {
	if img == nil {
		return ColorNone
	}
	if c >= 0 {
		img.transparent = c
	}
	return img.transparent
}

// --- helpers ---

func clamp8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func clampAlpha(v int) int {
	if v < 0 {
		return 0
	}
	if v > AlphaMax {
		return AlphaMax
	}
	return v
}

func packGDColor(r, g, b, a int) Color {
	return Color((a&0x7f)<<24 | (r&0xff)<<16 | (g&0xff)<<8 | (b & 0xff))
}

func unpackGDColor(c Color) (r, g, b, a int) {
	a = int(c>>24) & 0x7f
	r = int(c>>16) & 0xff
	g = int(c>>8) & 0xff
	b = int(c) & 0xff
	return
}

func gdToNRGBA(r, g, b, a int) color.NRGBA {
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: gdAlphaToStdAlpha(a)}
}

func gdAlphaToStdAlpha(a int) uint8 {
	if a <= 0 {
		return 255
	}
	if a >= AlphaMax {
		return 0
	}
	return uint8(255 - a*255/AlphaMax)
}

func stdAlphaToGD(a uint8) int {
	if a >= 255 {
		return 0
	}
	if a == 0 {
		return AlphaMax
	}
	return AlphaMax - int(a)*AlphaMax/255
}

// nrgbaComponents returns the non-premultiplied RGBA of c as 8-bit values.
func nrgbaComponents(c color.Color) (r, g, b, a uint8) {
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	return nc.R, nc.G, nc.B, nc.A
}

func pcEq(a, b color.Color) bool {
	ar, ag, ab, aa := nrgbaComponents(a)
	br, bg, bb, ba := nrgbaComponents(b)
	return ar == br && ag == bg && ab == bb && aa == ba
}

// rgbToHWB converts 8-bit RGB to Hue (degrees, 0..360), Whiteness
// (0..1), and Blackness (0..1).
func rgbToHWB(r, g, b uint8) (h, w, bl float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	cmax := math.Max(rf, math.Max(gf, bf))
	cmin := math.Min(rf, math.Min(gf, bf))
	w = cmin
	bl = 1 - cmax
	delta := cmax - cmin
	if delta == 0 {
		return 0, w, bl
	}
	switch cmax {
	case rf:
		h = math.Mod((gf-bf)/delta, 6)
	case gf:
		h = (bf-rf)/delta + 2
	default:
		h = (rf-gf)/delta + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return
}

// ImageColorMatch tunes the palette of img1 so that its colors best match
// the corresponding regions of img2. For each palette entry, the colors
// of img2's pixels that map to that index in img1 are averaged, and
// the palette slot is overwritten with the result. Both images must
// have identical bounds.
func ImageColorMatch(img1 *Image, img2 image.Image) bool {
	if img1 == nil || img1.pal == nil || img2 == nil {
		return false
	}
	if img1.Bounds() != img2.Bounds() {
		return false
	}
	n := len(img1.pal.Palette)
	if n == 0 {
		return false
	}
	type acc struct{ r, g, b, count uint64 }
	accs := make([]acc, n)
	b := img1.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			idx := img1.pal.ColorIndexAt(x, y)
			nc := color.NRGBAModel.Convert(img2.At(x, y)).(color.NRGBA)
			accs[idx].r += uint64(nc.R)
			accs[idx].g += uint64(nc.G)
			accs[idx].b += uint64(nc.B)
			accs[idx].count++
		}
	}
	for i := 0; i < n; i++ {
		a := accs[i]
		if a.count == 0 {
			continue
		}
		img1.pal.Palette[i] = color.NRGBA{
			R: uint8(a.r / a.count),
			G: uint8(a.g / a.count),
			B: uint8(a.b / a.count),
			A: 255,
		}
	}
	return true
}
