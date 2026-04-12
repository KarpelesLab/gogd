package gogd

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// --- copy ---

// ImageCopy copies a srcW×srcH rectangle from src at (srcX, srcY) to dst
// at (dstX, dstY). Accepts any [draw.Image] / [image.Image].
func ImageCopy(dstImg draw.Image, srcImg image.Image, dstX, dstY, srcX, srcY, srcW, srcH int) bool {
	if dstImg == nil || srcImg == nil {
		return false
	}
	dst := asImage(dstImg)
	if dst == nil {
		return false
	}
	dr := image.Rect(dstX, dstY, dstX+srcW, dstY+srcH)
	sp := image.Pt(srcX, srcY)
	if dst.nrgba != nil {
		op := draw.Over
		if !dst.alphaBlending {
			op = draw.Src
		}
		draw.Draw(dst.nrgba, dr, srcImg, sp, op)
		return true
	}
	if dst.pal != nil {
		for y := 0; y < srcH; y++ {
			for x := 0; x < srcW; x++ {
				c := srcImg.At(srcX+x, srcY+y)
				idx := dst.pal.Palette.Index(c)
				px, py := dstX+x, dstY+y
				if (image.Point{X: px, Y: py}).In(dst.Bounds()) {
					dst.pal.SetColorIndex(px, py, uint8(idx))
				}
			}
		}
		return true
	}
	if dst.generic != nil {
		op := draw.Over
		if !dst.alphaBlending {
			op = draw.Src
		}
		draw.Draw(dst.generic, dr, srcImg, sp, op)
		return true
	}
	return false
}

// ImageCopyMerge copies a rectangle from src to dst with a merge
// percentage (0 = no change, 100 = full src copy).
func ImageCopyMerge(dstImg draw.Image, srcImg image.Image, dstX, dstY, srcX, srcY, srcW, srcH, pct int) bool {
	if dstImg == nil || srcImg == nil || pct <= 0 {
		return false
	}
	dst := asImage(dstImg)
	if dst == nil {
		return false
	}
	if pct >= 100 {
		return ImageCopy(dstImg, srcImg, dstX, dstY, srcX, srcY, srcW, srcH)
	}
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			dx, dy := dstX+x, dstY+y
			if !(image.Point{X: dx, Y: dy}).In(dst.Bounds()) {
				continue
			}
			sc := nrgbaOf(srcImg.At(srcX+x, srcY+y))
			dc := nrgbaOf(dst.At(dx, dy))
			rc := color.NRGBA{
				R: uint8((int(sc.R)*pct + int(dc.R)*(100-pct)) / 100),
				G: uint8((int(sc.G)*pct + int(dc.G)*(100-pct)) / 100),
				B: uint8((int(sc.B)*pct + int(dc.B)*(100-pct)) / 100),
				A: uint8((int(sc.A)*pct + int(dc.A)*(100-pct)) / 100),
			}
			setNRGBAPixel(dst, dx, dy, rc)
		}
	}
	return true
}

// ImageCopyMergeGray is like ImageCopyMerge but converts each
// destination pixel to gray before blending, preserving the source hue.
func ImageCopyMergeGray(dstImg draw.Image, srcImg image.Image, dstX, dstY, srcX, srcY, srcW, srcH, pct int) bool {
	if dstImg == nil || srcImg == nil || pct <= 0 {
		return false
	}
	dst := asImage(dstImg)
	if dst == nil {
		return false
	}
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			dx, dy := dstX+x, dstY+y
			if !(image.Point{X: dx, Y: dy}).In(dst.Bounds()) {
				continue
			}
			sc := nrgbaOf(srcImg.At(srcX+x, srcY+y))
			dc := nrgbaOf(dst.At(dx, dy))
			gray := (int(dc.R) + int(dc.G) + int(dc.B)) / 3
			rc := color.NRGBA{
				R: uint8((int(sc.R)*pct + gray*(100-pct)) / 100),
				G: uint8((int(sc.G)*pct + gray*(100-pct)) / 100),
				B: uint8((int(sc.B)*pct + gray*(100-pct)) / 100),
				A: dc.A,
			}
			setNRGBAPixel(dst, dx, dy, rc)
		}
	}
	return true
}

// --- resize ---

// ImageCopyResized copies and resizes a rectangle from src to dst using
// nearest-neighbour interpolation.
func ImageCopyResized(dstImg draw.Image, srcImg image.Image, dstX, dstY, srcX, srcY, dstW, dstH, srcW, srcH int) bool {
	if dstImg == nil || srcImg == nil {
		return false
	}
	dr := image.Rect(dstX, dstY, dstX+dstW, dstY+dstH)
	sr := image.Rect(srcX, srcY, srcX+srcW, srcY+srcH)
	dstDraw := drawTarget(dstImg)
	if dstDraw == nil {
		return false
	}
	xdraw.NearestNeighbor.Scale(dstDraw, dr, underlyingImage(srcImg), sr, xdraw.Over, nil)
	return true
}

// ImageCopyResampled copies and resizes a rectangle from src to dst using
// high-quality bicubic interpolation.
func ImageCopyResampled(dstImg draw.Image, srcImg image.Image, dstX, dstY, srcX, srcY, dstW, dstH, srcW, srcH int) bool {
	if dstImg == nil || srcImg == nil {
		return false
	}
	dr := image.Rect(dstX, dstY, dstX+dstW, dstY+dstH)
	sr := image.Rect(srcX, srcY, srcX+srcW, srcY+srcH)
	dstDraw := drawTarget(dstImg)
	if dstDraw == nil {
		return false
	}
	xdraw.CatmullRom.Scale(dstDraw, dr, underlyingImage(srcImg), sr, xdraw.Over, nil)
	return true
}

// Interpolation mode constants matching PHP's IMG_* flags.
const (
	ImgNearestNeighbour = 0
	ImgBilinearFixed    = 1
	ImgBicubic          = 2
	ImgBicubicFixed     = 3
)

// ImageScale returns a new truecolor image scaled to newW × newH. Pass
// -1 for newH to preserve the aspect ratio. mode selects the
// interpolation algorithm. Accepts any [image.Image].
func ImageScale(src image.Image, newW, newH, mode int) *Image {
	if src == nil || newW <= 0 {
		return nil
	}
	b := src.Bounds()
	if newH <= 0 {
		newH = b.Dy() * newW / b.Dx()
		if newH <= 0 {
			newH = 1
		}
	}
	dst := ImageCreateTrueColor(newW, newH)
	var interp xdraw.Interpolator
	switch mode {
	case ImgBicubic, ImgBicubicFixed:
		interp = xdraw.CatmullRom
	case ImgBilinearFixed:
		interp = xdraw.BiLinear
	default:
		interp = xdraw.NearestNeighbor
	}
	interp.Scale(dst.nrgba, dst.Bounds(), underlyingImage(src), b, xdraw.Over, nil)
	return dst
}

// --- flip ---

// Flip mode constants matching PHP.
const (
	ImgFlipHorizontal = 1
	ImgFlipVertical   = 2
	ImgFlipBoth       = 3
)

// ImageFlip flips img in-place according to mode. Accepts any
// [draw.Image].
func ImageFlip(dst draw.Image, mode int) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if img.nrgba != nil {
		m := img.nrgba
		if mode&ImgFlipHorizontal != 0 {
			for y := 0; y < h; y++ {
				for x := 0; x < w/2; x++ {
					rx := w - 1 - x
					l, r := m.NRGBAAt(x, y), m.NRGBAAt(rx, y)
					m.SetNRGBA(x, y, r)
					m.SetNRGBA(rx, y, l)
				}
			}
		}
		if mode&ImgFlipVertical != 0 {
			for y := 0; y < h/2; y++ {
				ry := h - 1 - y
				for x := 0; x < w; x++ {
					top, bot := m.NRGBAAt(x, y), m.NRGBAAt(x, ry)
					m.SetNRGBA(x, y, bot)
					m.SetNRGBA(x, ry, top)
				}
			}
		}
		return true
	}
	if img.pal != nil {
		m := img.pal
		if mode&ImgFlipHorizontal != 0 {
			for y := 0; y < h; y++ {
				for x := 0; x < w/2; x++ {
					rx := w - 1 - x
					l, r := m.ColorIndexAt(x, y), m.ColorIndexAt(rx, y)
					m.SetColorIndex(x, y, r)
					m.SetColorIndex(rx, y, l)
				}
			}
		}
		if mode&ImgFlipVertical != 0 {
			for y := 0; y < h/2; y++ {
				ry := h - 1 - y
				for x := 0; x < w; x++ {
					top, bot := m.ColorIndexAt(x, y), m.ColorIndexAt(x, ry)
					m.SetColorIndex(x, y, bot)
					m.SetColorIndex(x, ry, top)
				}
			}
		}
		return true
	}
	if img.generic != nil {
		m := img.generic
		if mode&ImgFlipHorizontal != 0 {
			for y := 0; y < h; y++ {
				for x := 0; x < w/2; x++ {
					rx := w - 1 - x
					l, r := m.At(x, y), m.At(rx, y)
					m.Set(x, y, r)
					m.Set(rx, y, l)
				}
			}
		}
		if mode&ImgFlipVertical != 0 {
			for y := 0; y < h/2; y++ {
				ry := h - 1 - y
				for x := 0; x < w; x++ {
					top, bot := m.At(x, y), m.At(x, ry)
					m.Set(x, y, bot)
					m.Set(x, ry, top)
				}
			}
		}
		return true
	}
	return false
}

// --- crop ---

// ImageCrop returns a new truecolor image cropped to the given rectangle.
// Accepts any [image.Image].
func ImageCrop(src image.Image, rect image.Rectangle) *Image {
	if src == nil {
		return nil
	}
	rect = rect.Intersect(src.Bounds())
	if rect.Empty() {
		return nil
	}
	dst := ImageCreateTrueColor(rect.Dx(), rect.Dy())
	ImageAlphaBlending(dst, false)
	ImageCopy(dst, src, 0, 0, rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
	ImageAlphaBlending(dst, true)
	return dst
}

// --- rotate ---

// ImageRotate returns a new truecolor image rotated counter-clockwise by
// angle degrees, with exposed background filled with bgColor. Sampling
// uses bilinear interpolation. Accepts any [image.Image].
func ImageRotate(src image.Image, angle float64, bgColor Color) *Image {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	wf, hf := float64(w), float64(h)

	rad := angle * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	if math.Abs(cos) < 1e-10 {
		cos = 0
	}
	if math.Abs(sin) < 1e-10 {
		sin = 0
	}

	nw := int(math.Ceil(math.Abs(wf*cos) + math.Abs(hf*sin)))
	nh := int(math.Ceil(math.Abs(wf*sin) + math.Abs(hf*cos)))
	if nw <= 0 {
		nw = 1
	}
	if nh <= 0 {
		nh = 1
	}

	dst := ImageCreateTrueColor(nw, nh)
	ImageAlphaBlending(dst, false)
	ImageFilledRectangle(dst, 0, 0, nw-1, nh-1, bgColor)

	// For each dst pixel, compute the source pixel via inverse rotation
	// (dst→src): rotate by -angle around dst centre and translate to
	// src centre.
	cx, cy := wf/2, hf/2
	ncx, ncy := float64(nw)/2, float64(nh)/2

	for dy := 0; dy < nh; dy++ {
		for dx := 0; dx < nw; dx++ {
			fx := float64(dx) + 0.5 - ncx
			fy := float64(dy) + 0.5 - ncy
			// Inverse map dst→src: rotate dst offset by -θ (i.e. CW by θ)
			// around the destination centre, then shift to the source centre.
			sx := cos*fx - sin*fy + cx
			sy := sin*fx + cos*fy + cy
			sxi := int(math.Floor(sx))
			syi := int(math.Floor(sy))
			if sxi < 0 || sxi >= w || syi < 0 || syi >= h {
				continue
			}
			dst.nrgba.SetNRGBA(dx, dy, nrgbaOf(src.At(sxi+b.Min.X, syi+b.Min.Y)))
		}
	}
	ImageAlphaBlending(dst, true)
	return dst
}

// --- helpers ---

func nrgbaOf(c color.Color) color.NRGBA {
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}

func setNRGBAPixel(img *Image, x, y int, c color.NRGBA) {
	if img.nrgba != nil {
		img.nrgba.SetNRGBA(x, y, c)
		return
	}
	if img.pal != nil {
		idx := img.pal.Palette.Index(c)
		img.pal.SetColorIndex(x, y, uint8(idx))
		return
	}
	if img.generic != nil {
		img.generic.Set(x, y, c)
	}
}

// drawTarget returns a [xdraw.Image] suitable for xdraw's fast paths.
// For our wrapper we unwrap to the underlying NRGBA/Paletted; for any
// other draw.Image we pass it through.
func drawTarget(dstImg draw.Image) xdraw.Image {
	if g, ok := dstImg.(*Image); ok {
		if g == nil {
			return nil
		}
		if g.nrgba != nil {
			return g.nrgba
		}
		if g.pal != nil {
			return g.pal
		}
		if g.generic != nil {
			return g.generic
		}
		return nil
	}
	return dstImg
}

// underlyingImage returns the concrete stdlib image backing src. xdraw's
// fast paths dispatch on specific image types, so passing our wrapper
// yields incorrect results.
func underlyingImage(src image.Image) image.Image {
	if g, ok := src.(*Image); ok {
		if g == nil {
			return nil
		}
		if g.nrgba != nil {
			return g.nrgba
		}
		if g.pal != nil {
			return g.pal
		}
		if g.generic != nil {
			return g.generic
		}
		return g
	}
	return src
}
