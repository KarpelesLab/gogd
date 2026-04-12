package gogd

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"math"
	"sort"
)

// Filter modes matching PHP's IMG_FILTER_* constants.
const (
	FilterNegate        = 0
	FilterGrayscale     = 1
	FilterBrightness    = 2
	FilterContrast      = 3
	FilterColorize      = 4
	FilterEdgeDetect    = 5
	FilterEmboss        = 6
	FilterGaussianBlur  = 7
	FilterSelectiveBlur = 8
	FilterMeanRemoval   = 9
	FilterSmooth        = 10
	FilterPixelate      = 11
	FilterScatter       = 12
)

// Layer effect constants matching PHP's IMG_EFFECT_* flags.
const (
	EffectReplace    = 0
	EffectAlphaBlend = 1
	EffectNormal     = 2
	EffectOverlay    = 3
	EffectMultiply   = 4
)

// ImageFilter applies a filter to img. Additional args are interpreted
// per filter, matching PHP imagefilter semantics.
func ImageFilter(img *Image, filter int, args ...int) bool {
	if img == nil {
		return false
	}
	switch filter {
	case FilterNegate:
		return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
			return color.NRGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: c.A}
		})
	case FilterGrayscale:
		return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
			y := uint8(math.Round(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)))
			return color.NRGBA{R: y, G: y, B: y, A: c.A}
		})
	case FilterBrightness:
		if len(args) < 1 {
			return false
		}
		level := args[0]
		return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
			return color.NRGBA{
				R: clampU8(float64(int(c.R) + level)),
				G: clampU8(float64(int(c.G) + level)),
				B: clampU8(float64(int(c.B) + level)),
				A: c.A,
			}
		})
	case FilterContrast:
		if len(args) < 1 {
			return false
		}
		// libgd: factor = (100 - arg) / 100; arg > 0 reduces contrast.
		factor := (100.0 - float64(args[0])) / 100.0
		return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
			return color.NRGBA{
				R: clampU8(((float64(c.R)/255-0.5)*factor + 0.5) * 255),
				G: clampU8(((float64(c.G)/255-0.5)*factor + 0.5) * 255),
				B: clampU8(((float64(c.B)/255-0.5)*factor + 0.5) * 255),
				A: c.A,
			}
		})
	case FilterColorize:
		if len(args) < 3 {
			return false
		}
		dr, dg, db := args[0], args[1], args[2]
		da := 0
		if len(args) >= 4 {
			da = args[3]
		}
		return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
			newA := stdAlphaToGD(c.A) + da
			return color.NRGBA{
				R: clampU8(float64(int(c.R) + dr)),
				G: clampU8(float64(int(c.G) + dg)),
				B: clampU8(float64(int(c.B) + db)),
				A: gdAlphaToStdAlpha(clampAlpha(newA)),
			}
		})
	case FilterEdgeDetect:
		return applyKernel(img, [3][3]float64{
			{-1, 0, -1},
			{0, 4, 0},
			{-1, 0, -1},
		}, 1, 0)
	case FilterEmboss:
		return applyKernel(img, [3][3]float64{
			{1.5, 0, 0},
			{0, 0, 0},
			{0, 0, -1.5},
		}, 1, 127)
	case FilterGaussianBlur:
		return applyKernel(img, [3][3]float64{
			{1, 2, 1},
			{2, 4, 2},
			{1, 2, 1},
		}, 16, 0)
	case FilterMeanRemoval:
		return applyKernel(img, [3][3]float64{
			{-1, -1, -1},
			{-1, 9, -1},
			{-1, -1, -1},
		}, 1, 0)
	case FilterSmooth:
		if len(args) < 1 {
			return false
		}
		weight := float64(args[0])
		return applyKernel(img, [3][3]float64{
			{1, 1, 1},
			{1, weight, 1},
			{1, 1, 1},
		}, weight+8, 0)
	case FilterPixelate:
		if len(args) < 1 {
			return false
		}
		blockSize := args[0]
		advanced := len(args) >= 2 && args[1] != 0
		return applyPixelate(img, blockSize, advanced)
	}
	return false
}

// ImageConvolution applies a 3×3 convolution matrix to img. The result
// of each pixel is divided by divisor and then offset is added.
func ImageConvolution(img *Image, matrix [3][3]float64, divisor, offset float64) bool {
	return applyKernel(img, matrix, divisor, offset)
}

// ImageGammaCorrect applies a gamma transform. The correction factor is
// inputGamma / outputGamma; values > 1 darken, values < 1 brighten.
func ImageGammaCorrect(img *Image, inputGamma, outputGamma float64) bool {
	if img == nil || inputGamma <= 0 || outputGamma <= 0 {
		return false
	}
	g := inputGamma / outputGamma
	var lut [256]uint8
	for i := 0; i < 256; i++ {
		lut[i] = clampU8(255 * math.Pow(float64(i)/255, g))
	}
	return perPixelFilter(img, func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{R: lut[c.R], G: lut[c.G], B: lut[c.B], A: c.A}
	})
}

// ImageLayerEffect selects the pixel-combination mode used by subsequent
// drawing operations. Currently EffectReplace maps to alphaBlending off
// and EffectAlphaBlend/EffectNormal to alphaBlending on; overlay and
// multiply are accepted but not yet implemented.
func ImageLayerEffect(img *Image, effect int) bool {
	if img == nil {
		return false
	}
	switch effect {
	case EffectReplace:
		img.alphaBlending = false
	case EffectAlphaBlend, EffectNormal:
		img.alphaBlending = true
	}
	return true
}

// ImageColorSet changes the RGB components of a palette entry.
func ImageColorSet(img *Image, index Color, r, g, b int) bool {
	return ImageColorSetAlpha(img, index, r, g, b, AlphaOpaque)
}

// ImageColorSetAlpha is like ImageColorSet but also updates alpha.
func ImageColorSetAlpha(img *Image, index Color, r, g, b, a int) bool {
	if img == nil || img.pal == nil {
		return false
	}
	if int(index) < 0 || int(index) >= len(img.pal.Palette) {
		return false
	}
	img.pal.Palette[int(index)] = gdToNRGBA(clamp8(r), clamp8(g), clamp8(b), clampAlpha(a))
	return true
}

// ImagePaletteCopy copies the palette from src to dst. Both images must
// be palette-backed.
func ImagePaletteCopy(dst, src *Image) bool {
	if dst == nil || src == nil || dst.pal == nil || src.pal == nil {
		return false
	}
	dst.pal.Palette = append(color.Palette{}, src.pal.Palette...)
	return true
}

// ImagePaletteToTrueColor converts a palette image to truecolor in place.
// Truecolor images are returned unchanged.
func ImagePaletteToTrueColor(img *Image) bool {
	if img == nil {
		return false
	}
	if img.nrgba != nil {
		return true
	}
	if img.pal == nil {
		return false
	}
	b := img.pal.Bounds()
	nrgba := image.NewNRGBA(b)
	draw.Draw(nrgba, b, img.pal, b.Min, draw.Src)
	img.pal = nil
	img.nrgba = nrgba
	return true
}

// ImageTrueColorToPalette converts a truecolor image to a palette image
// using the Plan9 palette truncated to numColors entries (1..256). When
// dither is true Floyd–Steinberg dithering is applied.
func ImageTrueColorToPalette(img *Image, dither bool, numColors int) bool {
	if img == nil {
		return false
	}
	if img.pal != nil {
		return true
	}
	if img.nrgba == nil {
		return false
	}
	if numColors < 1 {
		numColors = 1
	}
	if numColors > 256 {
		numColors = 256
	}
	p := buildPalette(img.nrgba, numColors)
	b := img.nrgba.Bounds()
	pm := image.NewPaletted(b, p)
	if dither {
		draw.FloydSteinberg.Draw(pm, b, img.nrgba, b.Min)
	} else {
		draw.Draw(pm, b, img.nrgba, b.Min, draw.Src)
	}
	img.nrgba = nil
	img.pal = pm
	return true
}

// --- internals ---

// buildPalette produces a color.Palette of at most n entries for m. If
// m has ≤ n unique colors, those are used verbatim. Otherwise the n
// most frequent colors are picked. This is a simple quantizer — a
// proper median-cut pass is left for a later milestone.
func buildPalette(m *image.NRGBA, n int) color.Palette {
	seen := make(map[color.NRGBA]int)
	for i := 0; i+3 < len(m.Pix); i += 4 {
		c := color.NRGBA{R: m.Pix[i], G: m.Pix[i+1], B: m.Pix[i+2], A: m.Pix[i+3]}
		seen[c]++
	}
	if len(seen) == 0 {
		return color.Palette{color.NRGBA{}}
	}
	if len(seen) <= n {
		p := make(color.Palette, 0, len(seen))
		for c := range seen {
			p = append(p, c)
		}
		return p
	}
	type entry struct {
		c color.NRGBA
		n int
	}
	entries := make([]entry, 0, len(seen))
	for c, count := range seen {
		entries = append(entries, entry{c, count})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].n > entries[j].n })
	p := make(color.Palette, 0, n)
	for i := 0; i < n; i++ {
		p = append(p, entries[i].c)
	}
	_ = palette.Plan9 // reserved for a future median-cut fallback
	return p
}

func perPixelFilter(img *Image, f func(color.NRGBA) color.NRGBA) bool {
	if img.nrgba != nil {
		pix := img.nrgba.Pix
		for i := 0; i+3 < len(pix); i += 4 {
			c := color.NRGBA{R: pix[i], G: pix[i+1], B: pix[i+2], A: pix[i+3]}
			r := f(c)
			pix[i], pix[i+1], pix[i+2], pix[i+3] = r.R, r.G, r.B, r.A
		}
		return true
	}
	if img.pal != nil {
		for i, c := range img.pal.Palette {
			img.pal.Palette[i] = f(nrgbaOf(c))
		}
		return true
	}
	return false
}

func applyKernel(img *Image, k [3][3]float64, divisor, offset float64) bool {
	if img == nil || img.nrgba == nil {
		return false
	}
	dst := img.nrgba
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(b)
	copy(src.Pix, dst.Pix)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum float64
			for ky := 0; ky < 3; ky++ {
				for kx := 0; kx < 3; kx++ {
					sx := x + kx - 1
					sy := y + ky - 1
					if sx < 0 {
						sx = 0
					}
					if sx >= w {
						sx = w - 1
					}
					if sy < 0 {
						sy = 0
					}
					if sy >= h {
						sy = h - 1
					}
					c := src.NRGBAAt(sx, sy)
					kw := k[ky][kx]
					rSum += float64(c.R) * kw
					gSum += float64(c.G) * kw
					bSum += float64(c.B) * kw
				}
			}
			if divisor != 0 {
				rSum /= divisor
				gSum /= divisor
				bSum /= divisor
			}
			rSum += offset
			gSum += offset
			bSum += offset
			origA := src.NRGBAAt(x, y).A
			dst.SetNRGBA(x, y, color.NRGBA{
				R: clampU8(rSum),
				G: clampU8(gSum),
				B: clampU8(bSum),
				A: origA,
			})
		}
	}
	return true
}

func applyPixelate(img *Image, blockSize int, advanced bool) bool {
	if img == nil || img.nrgba == nil || blockSize < 1 {
		return false
	}
	m := img.nrgba
	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	for by := 0; by < h; by += blockSize {
		for bx := 0; bx < w; bx += blockSize {
			ex := bx + blockSize
			if ex > w {
				ex = w
			}
			ey := by + blockSize
			if ey > h {
				ey = h
			}
			var rSum, gSum, bSum, aSum, count int
			if advanced {
				for y := by; y < ey; y++ {
					for x := bx; x < ex; x++ {
						c := m.NRGBAAt(x, y)
						rSum += int(c.R)
						gSum += int(c.G)
						bSum += int(c.B)
						aSum += int(c.A)
						count++
					}
				}
			} else {
				c := m.NRGBAAt(bx, by)
				rSum, gSum, bSum, aSum, count = int(c.R), int(c.G), int(c.B), int(c.A), 1
			}
			avg := color.NRGBA{
				R: uint8(rSum / count),
				G: uint8(gSum / count),
				B: uint8(bSum / count),
				A: uint8(aSum / count),
			}
			for y := by; y < ey; y++ {
				for x := bx; x < ex; x++ {
					m.SetNRGBA(x, y, avg)
				}
			}
		}
	}
	return true
}

func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}
