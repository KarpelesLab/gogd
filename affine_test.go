package gogd

import (
	"image"
	"math"
	"testing"
)

func TestInterpolationGetSet(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	if got := ImageGetInterpolation(img); got != ImgBilinearFixed {
		t.Errorf("default interpolation = %d", got)
	}
	ImageSetInterpolation(img, ImgBicubic)
	if got := ImageGetInterpolation(img); got != ImgBicubic {
		t.Errorf("after set = %d", got)
	}
}

func TestResolutionGetSet(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	x, y := ImageGetResolution(img)
	if x != 96 || y != 96 {
		t.Errorf("default resolution = %d, %d", x, y)
	}
	ImageResolution(img, 300, 300)
	x, y = ImageGetResolution(img)
	if x != 300 || y != 300 {
		t.Errorf("after set = %d, %d", x, y)
	}
	// Negative leaves axis unchanged.
	ImageResolution(img, -1, 150)
	x, y = ImageGetResolution(img)
	if x != 300 || y != 150 {
		t.Errorf("partial set = %d, %d", x, y)
	}
}

func TestAffineMatrixGet(t *testing.T) {
	m, err := ImageAffineMatrixGet(AffineTranslate, 5, 7)
	if err != nil {
		t.Fatal(err)
	}
	if m[4] != 5 || m[5] != 7 {
		t.Errorf("translate = %v", m)
	}
	m, err = ImageAffineMatrixGet(AffineScale, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if m[0] != 2 || m[3] != 3 {
		t.Errorf("scale = %v", m)
	}
	m, err = ImageAffineMatrixGet(AffineRotate, 90)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m[0]) > 1e-10 || math.Abs(m[1]-(-1)) > 1e-10 || math.Abs(m[2]-1) > 1e-10 {
		t.Errorf("rotate 90 = %v", m)
	}
}

func TestAffineMatrixGetMissingArgs(t *testing.T) {
	if _, err := ImageAffineMatrixGet(AffineTranslate, 1); err == nil {
		t.Error("missing tx/ty should error")
	}
}

func TestAffineMatrixConcat(t *testing.T) {
	trans, _ := ImageAffineMatrixGet(AffineTranslate, 10, 0)
	scale, _ := ImageAffineMatrixGet(AffineScale, 2, 2)
	// Apply scale then translate: (x, y) → (2x, 2y) → (2x + 10, 2y)
	c := ImageAffineMatrixConcat(scale, trans)
	// Check point (1, 1) → (2, 2) → (12, 2)
	x := c[0]*1 + c[2]*1 + c[4]
	y := c[1]*1 + c[3]*1 + c[5]
	if math.Abs(x-12) > 1e-10 || math.Abs(y-2) > 1e-10 {
		t.Errorf("concat transform = (%v, %v), want (12, 2)", x, y)
	}
}

func TestAffineScale(t *testing.T) {
	src := makeSolid(5, 5, 100, 200, 50)
	m, _ := ImageAffineMatrixGet(AffineScale, 2, 2)
	dst := ImageAffine(src, m, nil)
	if dst == nil {
		t.Fatal("affine returned nil")
	}
	if dst.Width() != 10 || dst.Height() != 10 {
		t.Errorf("scaled size = %dx%d", dst.Width(), dst.Height())
	}
	r, g, b, _ := ImageColorsForIndex(dst, ImageColorAt(dst, 5, 5))
	if r != 100 || g != 200 || b != 50 {
		t.Errorf("scaled pixel = %d,%d,%d", r, g, b)
	}
}

func TestCropAutoTransparent(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	// Start fully transparent.
	ImageAlphaBlending(img, false)
	transparent := ImageColorAllocateAlpha(img, 0, 0, 0, AlphaTransparent)
	ImageFilledRectangle(img, 0, 0, 9, 9, transparent)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 3, 3, 6, 6, red)

	dst := ImageCropAuto(img, CropTransparent, 0, 0)
	if dst == nil {
		t.Fatal("cropauto nil")
	}
	if dst.Width() != 4 || dst.Height() != 4 {
		t.Errorf("cropped size = %dx%d, want 4x4", dst.Width(), dst.Height())
	}
}

func TestCropAutoSides(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 9, 9, white)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 2, 2, 7, 7, red)

	dst := ImageCropAuto(img, CropSides, 0, 0)
	if dst == nil {
		t.Fatal("cropauto nil")
	}
	if dst.Width() != 6 || dst.Height() != 6 {
		t.Errorf("cropped = %dx%d, want 6x6", dst.Width(), dst.Height())
	}
}

func TestCropAutoEmpty(t *testing.T) {
	// Fully-matching image: nothing to keep.
	img := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(img, false)
	transparent := ImageColorAllocateAlpha(img, 0, 0, 0, AlphaTransparent)
	ImageFilledRectangle(img, 0, 0, 3, 3, transparent)
	if dst := ImageCropAuto(img, CropTransparent, 0, 0); dst != nil {
		t.Error("expected nil for fully-transparent input")
	}
}

func TestCropAutoThreshold(t *testing.T) {
	img := ImageCreateTrueColor(6, 6)
	ImageAlphaBlending(img, false)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 5, 5, white)
	// Near-white core (still passes threshold=10)
	ImageFilledRectangle(img, 0, 0, 5, 5, ImageColorAllocate(img, 254, 254, 254))
	// A clearly different red blob:
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 2, 2, 3, 3, red)

	dst := ImageCropAuto(img, CropThreshold, 10, white)
	if dst == nil {
		t.Fatal("nil dst")
	}
	if dst.Width() != 2 || dst.Height() != 2 {
		t.Errorf("threshold crop = %dx%d, want 2x2", dst.Width(), dst.Height())
	}
}

func TestAffineDegenerate(t *testing.T) {
	// Zero determinant matrix: should return nil.
	img := makeSolid(2, 2, 0, 0, 0)
	m := [6]float64{0, 0, 0, 0, 0, 0}
	if ImageAffine(img, m, nil) != nil {
		t.Error("expected nil for singular matrix")
	}
}

// Sanity-check that image.Rect intersection still works on Image.
func TestAffineWithClip(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 9, 9, white)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 0, 0, 4, 4, red)

	m, _ := ImageAffineMatrixGet(AffineScale, 1, 1)
	clip := image.Rect(0, 0, 5, 5)
	dst := ImageAffine(img, m, &clip)
	if dst == nil {
		t.Fatal("nil dst")
	}
	if dst.Width() != 5 || dst.Height() != 5 {
		t.Errorf("clipped affine = %dx%d, want 5x5", dst.Width(), dst.Height())
	}
	r, _, _, _ := ImageColorsForIndex(dst, ImageColorAt(dst, 0, 0))
	if r != 255 {
		t.Error("clipped pixel not red")
	}
}
