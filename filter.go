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
// per filter, matching PHP imagefilter semantics. Accepts any
// [draw.Image]; kernel-based filters require a direct NRGBA buffer
// (our truecolor mode or *image.NRGBA).
func ImageFilter(dst draw.Image, filter int, args ...int) bool {
	img := asImage(dst)
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
	case FilterSelectiveBlur:
		return applySelectiveBlur(img)
	case FilterScatter:
		if len(args) < 2 {
			return false
		}
		return applyScatter(img, args[0], args[1])
	}
	return false
}

// ImageConvolution applies a 3×3 convolution matrix to img. The result
// of each pixel is divided by divisor and then offset is added. Accepts
// any [draw.Image].
func ImageConvolution(dst draw.Image, matrix [3][3]float64, divisor, offset float64) bool {
	img := asImage(dst)
	return applyKernel(img, matrix, divisor, offset)
}

// ImageGammaCorrect applies a gamma transform. The correction factor is
// inputGamma / outputGamma; values > 1 darken, values < 1 brighten.
// Accepts any [draw.Image].
func ImageGammaCorrect(dst draw.Image, inputGamma, outputGamma float64) bool {
	img := asImage(dst)
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
// m has ≤ n unique colors, those are used verbatim. Otherwise median-cut
// recursively partitions the color space along its widest channel until
// n buckets remain, and each bucket contributes its weighted-mean color.
func buildPalette(m *image.NRGBA, n int) color.Palette {
	seen := make(map[color.NRGBA]uint32)
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
	return medianCutPalette(seen, n)
}

type mcEntry struct {
	c color.NRGBA
	w uint32
}

type mcBucket struct{ start, end int }

// medianCutPalette quantises the given weighted color histogram down to
// n representative entries using the classic median-cut algorithm.
func medianCutPalette(hist map[color.NRGBA]uint32, n int) color.Palette {
	pool := make([]mcEntry, 0, len(hist))
	for c, w := range hist {
		pool = append(pool, mcEntry{c, w})
	}
	buckets := []mcBucket{{0, len(pool)}}
	for len(buckets) < n {
		bi, ch := pickSplitBucket(pool, buckets)
		if bi < 0 {
			break
		}
		b := buckets[bi]
		slice := pool[b.start:b.end]
		sort.Slice(slice, func(i, j int) bool {
			return channelValue(slice[i].c, ch) < channelValue(slice[j].c, ch)
		})
		mid := b.start + len(slice)/2
		buckets[bi] = mcBucket{b.start, mid}
		buckets = append(buckets, mcBucket{mid, b.end})
	}
	p := make(color.Palette, 0, len(buckets))
	for _, b := range buckets {
		if b.end <= b.start {
			continue
		}
		var sr, sg, sb, sa, total uint64
		for k := b.start; k < b.end; k++ {
			e := pool[k]
			w := uint64(e.w)
			sr += uint64(e.c.R) * w
			sg += uint64(e.c.G) * w
			sb += uint64(e.c.B) * w
			sa += uint64(e.c.A) * w
			total += w
		}
		if total == 0 {
			continue
		}
		p = append(p, color.NRGBA{
			R: uint8(sr / total),
			G: uint8(sg / total),
			B: uint8(sb / total),
			A: uint8(sa / total),
		})
	}
	_ = palette.Plan9 // keep the import in case we fall back here later
	return p
}

// pickSplitBucket returns the index of the bucket with the largest
// range in any single channel, and which channel (0=R, 1=G, 2=B) spans
// that range. Returns -1 when no bucket can be split further.
func pickSplitBucket(pool []mcEntry, buckets []mcBucket) (int, int) {
	bestBucket, bestCh, bestRange := -1, 0, -1
	for i, b := range buckets {
		if b.end-b.start < 2 {
			continue
		}
		var mn, mx [3]int
		for j := 0; j < 3; j++ {
			mn[j], mx[j] = 256, -1
		}
		for k := b.start; k < b.end; k++ {
			vs := [3]int{int(pool[k].c.R), int(pool[k].c.G), int(pool[k].c.B)}
			for j := 0; j < 3; j++ {
				if vs[j] < mn[j] {
					mn[j] = vs[j]
				}
				if vs[j] > mx[j] {
					mx[j] = vs[j]
				}
			}
		}
		for j := 0; j < 3; j++ {
			r := mx[j] - mn[j]
			if r > bestRange {
				bestRange, bestBucket, bestCh = r, i, j
			}
		}
	}
	return bestBucket, bestCh
}

func channelValue(c color.NRGBA, ch int) uint8 {
	switch ch {
	case 0:
		return c.R
	case 1:
		return c.G
	}
	return c.B
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

// applySelectiveBlur averages each pixel with its 3×3 neighbours,
// excluding neighbours whose luminance differs from the centre by more
// than a fixed threshold. Preserves edges while smoothing flat areas.
func applySelectiveBlur(img *Image) bool {
	if img == nil || img.nrgba == nil {
		return false
	}
	dst := img.nrgba
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(b)
	copy(src.Pix, dst.Pix)
	const threshold = 16
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.NRGBAAt(x, y)
			cl := int(c.R)*299 + int(c.G)*587 + int(c.B)*114
			var sr, sg, sb, count int
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					nc := src.NRGBAAt(nx, ny)
					nl := int(nc.R)*299 + int(nc.G)*587 + int(nc.B)*114
					d := nl - cl
					if d < 0 {
						d = -d
					}
					if d > threshold*1000 {
						continue
					}
					sr += int(nc.R)
					sg += int(nc.G)
					sb += int(nc.B)
					count++
				}
			}
			if count == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(sr / count),
				G: uint8(sg / count),
				B: uint8(sb / count),
				A: c.A,
			})
		}
	}
	return true
}

// applyScatter displaces each pixel by a random offset in the range
// [-sub, +plus] along each axis. Simulates film-grain / noise.
func applyScatter(img *Image, sub, plus int) bool {
	if img == nil || img.nrgba == nil {
		return false
	}
	if sub < 0 {
		sub = 0
	}
	if plus < 0 {
		plus = 0
	}
	span := sub + plus + 1
	if span <= 1 {
		return true
	}
	dst := img.nrgba
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(b)
	copy(src.Pix, dst.Pix)
	rng := scatterRNG()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ox := (rng() % uint32(span)) - uint32(sub)
			oy := (rng() % uint32(span)) - uint32(sub)
			nx := x + int(int32(ox))
			ny := y + int(int32(oy))
			if nx < 0 {
				nx = 0
			}
			if nx >= w {
				nx = w - 1
			}
			if ny < 0 {
				ny = 0
			}
			if ny >= h {
				ny = h - 1
			}
			dst.SetNRGBA(x, y, src.NRGBAAt(nx, ny))
		}
	}
	return true
}

// scatterRNG returns a simple xorshift32 PRNG. The seed is derived from
// a fixed constant so results are deterministic per call graph; the
// filter is aesthetic, so reproducibility aids testing.
func scatterRNG() func() uint32 {
	state := uint32(0x9e3779b9)
	return func() uint32 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return state
	}
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
