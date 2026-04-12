package gogd

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Affine-matrix types matching PHP's IMG_AFFINE_* constants.
const (
	AffineTranslate       = 0
	AffineScale           = 1
	AffineRotate          = 2
	AffineShearHorizontal = 3
	AffineShearVertical   = 4
)

// Crop-auto mode constants matching PHP's IMG_CROP_* flags.
const (
	CropDefault     = 0
	CropTransparent = 1
	CropBlack       = 2
	CropWhite       = 3
	CropSides       = 4
	CropThreshold   = 5
)

// --- interpolation ---

// ImageSetInterpolation records the interpolation method to use for
// future resampling operations. Returns false for nil images.
func ImageSetInterpolation(img *Image, method int) bool {
	if img == nil {
		return false
	}
	img.interpolation = method
	return true
}

// ImageGetInterpolation returns the current interpolation method.
func ImageGetInterpolation(img *Image) int {
	if img == nil {
		return 0
	}
	return img.interpolation
}

// --- resolution ---

// ImageResolution sets the horizontal and vertical resolution (DPI) for
// img. Pass -1 for either axis to leave it unchanged.
func ImageResolution(img *Image, resX, resY int) bool {
	if img == nil {
		return false
	}
	if resX >= 0 {
		img.resolutionX = resX
	}
	if resY >= 0 {
		img.resolutionY = resY
	}
	return true
}

// ImageGetResolution returns the horizontal and vertical DPI.
func ImageGetResolution(img *Image) (int, int) {
	if img == nil {
		return 0, 0
	}
	return img.resolutionX, img.resolutionY
}

// --- affine ---

// ImageAffineMatrixGet returns a 6-element affine matrix for the given
// transform type. The element order is PHP's (a, b, c, d, e, f) with
// x' = a·x + c·y + e, y' = b·x + d·y + f.
//
// Opts per type:
//   - AffineTranslate:      tx, ty
//   - AffineScale:          sx, sy
//   - AffineRotate:         angle in degrees
//   - AffineShearHorizontal, AffineShearVertical: angle in degrees
func ImageAffineMatrixGet(typ int, opts ...float64) ([6]float64, error) {
	need := func(n int, label string) error {
		if len(opts) < n {
			return fmt.Errorf("gogd: %s requires %d argument(s)", label, n)
		}
		return nil
	}
	switch typ {
	case AffineTranslate:
		if err := need(2, "translate"); err != nil {
			return [6]float64{}, err
		}
		return [6]float64{1, 0, 0, 1, opts[0], opts[1]}, nil
	case AffineScale:
		if err := need(2, "scale"); err != nil {
			return [6]float64{}, err
		}
		return [6]float64{opts[0], 0, 0, opts[1], 0, 0}, nil
	case AffineRotate:
		if err := need(1, "rotate"); err != nil {
			return [6]float64{}, err
		}
		rad := opts[0] * math.Pi / 180
		cos, sin := math.Cos(rad), math.Sin(rad)
		return [6]float64{cos, -sin, sin, cos, 0, 0}, nil
	case AffineShearHorizontal:
		if err := need(1, "shear"); err != nil {
			return [6]float64{}, err
		}
		t := math.Tan(opts[0] * math.Pi / 180)
		return [6]float64{1, 0, t, 1, 0, 0}, nil
	case AffineShearVertical:
		if err := need(1, "shear"); err != nil {
			return [6]float64{}, err
		}
		t := math.Tan(opts[0] * math.Pi / 180)
		return [6]float64{1, t, 0, 1, 0, 0}, nil
	}
	return [6]float64{}, fmt.Errorf("gogd: unknown affine type %d", typ)
}

// ImageAffineMatrixConcat concatenates two affine matrices: the returned
// matrix, applied to a point, is equivalent to first applying m1 then m2
// (matching PHP's imageaffinematrixconcat ordering).
func ImageAffineMatrixConcat(m1, m2 [6]float64) [6]float64 {
	return [6]float64{
		m1[0]*m2[0] + m1[1]*m2[2],
		m1[0]*m2[1] + m1[1]*m2[3],
		m1[2]*m2[0] + m1[3]*m2[2],
		m1[2]*m2[1] + m1[3]*m2[3],
		m1[4]*m2[0] + m1[5]*m2[2] + m2[4],
		m1[4]*m2[1] + m1[5]*m2[3] + m2[5],
	}
}

// ImageAffine applies the given affine matrix to img and returns a new
// truecolor image sized to the transformed bounding box. If clip is
// non-nil it bounds the source region considered.
func ImageAffine(img *Image, m [6]float64, clip *image.Rectangle) *Image {
	if img == nil {
		return nil
	}
	srcRect := img.Bounds()
	if clip != nil {
		srcRect = clip.Intersect(srcRect)
		if srcRect.Empty() {
			return nil
		}
	}
	w, h := srcRect.Dx(), srcRect.Dy()

	// Transform the four source-rect corners to compute dst size.
	apply := func(x, y float64) (float64, float64) {
		return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
	}
	corners := [4][2]float64{
		{0, 0},
		{float64(w), 0},
		{0, float64(h)},
		{float64(w), float64(h)},
	}
	var minX, minY, maxX, maxY float64
	for i, c := range corners {
		tx, ty := apply(c[0], c[1])
		if i == 0 {
			minX, minY, maxX, maxY = tx, tx, ty, ty
			minX, maxX = tx, tx
			minY, maxY = ty, ty
			continue
		}
		if tx < minX {
			minX = tx
		}
		if tx > maxX {
			maxX = tx
		}
		if ty < minY {
			minY = ty
		}
		if ty > maxY {
			maxY = ty
		}
	}
	nw := int(math.Ceil(maxX - minX))
	nh := int(math.Ceil(maxY - minY))
	if nw <= 0 {
		nw = 1
	}
	if nh <= 0 {
		nh = 1
	}

	// Inverse matrix so we can map each dst pixel back to src.
	det := m[0]*m[3] - m[1]*m[2]
	if math.Abs(det) < 1e-12 {
		return nil
	}
	inv := [6]float64{
		m[3] / det,
		-m[1] / det,
		-m[2] / det,
		m[0] / det,
		(m[2]*m[5] - m[3]*m[4]) / det,
		(m[1]*m[4] - m[0]*m[5]) / det,
	}

	dst := ImageCreateTrueColor(nw, nh)
	ImageAlphaBlending(dst, false)
	transparent := ImageColorAllocateAlpha(dst, 0, 0, 0, AlphaTransparent)
	ImageFilledRectangle(dst, 0, 0, nw-1, nh-1, transparent)

	for dy := 0; dy < nh; dy++ {
		for dx := 0; dx < nw; dx++ {
			fx := float64(dx) + minX
			fy := float64(dy) + minY
			sx := inv[0]*fx + inv[2]*fy + inv[4]
			sy := inv[1]*fx + inv[3]*fy + inv[5]
			sxi := int(math.Floor(sx))
			syi := int(math.Floor(sy))
			if sxi < 0 || sxi >= w || syi < 0 || syi >= h {
				continue
			}
			dst.nrgba.SetNRGBA(dx, dy, nrgbaOf(img.At(srcRect.Min.X+sxi, srcRect.Min.Y+syi)))
		}
	}
	ImageAlphaBlending(dst, true)
	return dst
}

// --- cropauto ---

// ImageCropAuto automatically crops img according to mode, using
// threshold and color when CropThreshold is requested. Returns a new
// truecolor image, or nil if the whole image matched the crop condition
// (no content left).
func ImageCropAuto(img *Image, mode int, threshold float64, c Color) *Image {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil
	}

	var matches func(x, y int) bool
	switch mode {
	case CropDefault:
		return ImageCropAuto(img, CropTransparent, threshold, c)
	case CropTransparent:
		matches = func(x, y int) bool {
			return nrgbaOf(img.At(x, y)).A == 0
		}
	case CropBlack:
		matches = func(x, y int) bool {
			nc := nrgbaOf(img.At(x, y))
			return nc.R == 0 && nc.G == 0 && nc.B == 0
		}
	case CropWhite:
		matches = func(x, y int) bool {
			nc := nrgbaOf(img.At(x, y))
			return nc.R == 255 && nc.G == 255 && nc.B == 255
		}
	case CropSides:
		// Use the four corner pixels' majority colour as the crop target.
		target := majorityCorner(img)
		matches = func(x, y int) bool {
			return sameColor(nrgbaOf(img.At(x, y)), target)
		}
	case CropThreshold:
		target := gdColorToNRGBA(c)
		matches = func(x, y int) bool {
			return colorDistance(nrgbaOf(img.At(x, y)), target) <= threshold
		}
	default:
		return nil
	}

	left, right := w, -1
	top, bottom := h, -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !matches(x, y) {
				if x < left {
					left = x
				}
				if x > right {
					right = x
				}
				if y < top {
					top = y
				}
				if y > bottom {
					bottom = y
				}
			}
		}
	}
	if right < left || bottom < top {
		return nil
	}
	return ImageCrop(img, image.Rect(left, top, right+1, bottom+1))
}

func majorityCorner(img *Image) color.NRGBA {
	b := img.Bounds()
	cs := []color.NRGBA{
		nrgbaOf(img.At(b.Min.X, b.Min.Y)),
		nrgbaOf(img.At(b.Max.X-1, b.Min.Y)),
		nrgbaOf(img.At(b.Min.X, b.Max.Y-1)),
		nrgbaOf(img.At(b.Max.X-1, b.Max.Y-1)),
	}
	counts := map[color.NRGBA]int{}
	for _, c := range cs {
		counts[c]++
	}
	var best color.NRGBA
	bestN := -1
	for c, n := range counts {
		if n > bestN {
			bestN, best = n, c
		}
	}
	return best
}

func sameColor(a, b color.NRGBA) bool {
	return a == b
}

func colorDistance(a, b color.NRGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

