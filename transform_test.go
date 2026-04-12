package gogd

import (
	"image"
	"testing"
)

func colorAt(t *testing.T, img *Image, x, y int) (r, g, b, a int) {
	t.Helper()
	return ImageColorsForIndex(img, ImageColorAt(img, x, y))
}

func TestImageCopyTrueColor(t *testing.T) {
	src := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(src, false)
	sc := ImageColorAllocate(src, 100, 150, 200)
	ImageFilledRectangle(src, 0, 0, 9, 9, sc)

	dst := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(dst, false)
	if !ImageCopy(dst, src, 5, 5, 0, 0, 10, 10) {
		t.Fatal("copy failed")
	}
	if r, g, b, _ := colorAt(t, dst, 10, 10); r != 100 || g != 150 || b != 200 {
		t.Errorf("copied pixel = %d,%d,%d", r, g, b)
	}
	if r, _, _, _ := colorAt(t, dst, 0, 0); r != 0 {
		t.Error("outside copy region should be unchanged")
	}
}

func TestImageCopyMerge(t *testing.T) {
	dst := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(dst, false)
	black := ImageColorAllocate(dst, 0, 0, 0)
	ImageFilledRectangle(dst, 0, 0, 3, 3, black)

	src := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(src, false)
	red := ImageColorAllocate(src, 255, 0, 0)
	ImageFilledRectangle(src, 0, 0, 3, 3, red)

	// 50% merge of red over black → (127, 0, 0)
	if !ImageCopyMerge(dst, src, 0, 0, 0, 0, 4, 4, 50) {
		t.Fatal("merge failed")
	}
	r, g, b, _ := colorAt(t, dst, 1, 1)
	if r < 100 || r > 140 || g != 0 || b != 0 {
		t.Errorf("50%% merge = %d,%d,%d, want ~127,0,0", r, g, b)
	}
}

func TestImageCopyMergeGray(t *testing.T) {
	dst := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(dst, false)
	// Destination pure red → gray = 255/3 = 85
	red := ImageColorAllocate(dst, 255, 0, 0)
	ImageFilledRectangle(dst, 0, 0, 3, 3, red)

	src := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(src, false)
	blue := ImageColorAllocate(src, 0, 0, 255)
	ImageFilledRectangle(src, 0, 0, 3, 3, blue)

	// 100% merge with gray: result should be fully blue (gray weight = 0)
	if !ImageCopyMergeGray(dst, src, 0, 0, 0, 0, 4, 4, 100) {
		t.Fatal("merge gray failed")
	}
	r, g, b, _ := colorAt(t, dst, 1, 1)
	if r != 0 || g != 0 || b != 255 {
		t.Errorf("full merge = %d,%d,%d, want 0,0,255", r, g, b)
	}
}

func TestImageCopyResized(t *testing.T) {
	src := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(src, false)
	c := ImageColorAllocate(src, 255, 0, 0)
	ImageFilledRectangle(src, 0, 0, 3, 3, c)

	dst := ImageCreateTrueColor(16, 16)
	ImageAlphaBlending(dst, false)
	if !ImageCopyResized(dst, src, 0, 0, 0, 0, 16, 16, 4, 4) {
		t.Fatal("resize failed")
	}
	if r, _, _, _ := colorAt(t, dst, 8, 8); r != 255 {
		t.Errorf("resized pixel not red: r=%d", r)
	}
}

func TestImageScale(t *testing.T) {
	src := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(src, false)
	c := ImageColorAllocate(src, 0, 255, 0)
	ImageFilledRectangle(src, 0, 0, 9, 9, c)

	// Scale with aspect preserved
	dst := ImageScale(src, 20, -1, ImgBicubic)
	if dst == nil {
		t.Fatal("scale returned nil")
	}
	if dst.Width() != 20 || dst.Height() != 20 {
		t.Errorf("aspect scale: got %dx%d", dst.Width(), dst.Height())
	}
	if _, g, _, _ := colorAt(t, dst, 10, 10); g < 200 {
		t.Errorf("scaled green = %d", g)
	}
}

func TestImageFlipHorizontal(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageSetPixel(img, 0, 0, red)
	ImageFlip(img, ImgFlipHorizontal)
	if r, _, _, _ := colorAt(t, img, 3, 0); r != 255 {
		t.Error("horizontal flip did not move pixel")
	}
	if r, _, _, _ := colorAt(t, img, 0, 0); r == 255 {
		t.Error("original pixel still present after horizontal flip")
	}
}

func TestImageFlipVertical(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageSetPixel(img, 0, 0, red)
	ImageFlip(img, ImgFlipVertical)
	if r, _, _, _ := colorAt(t, img, 0, 3); r != 255 {
		t.Error("vertical flip did not move pixel")
	}
}

func TestImageFlipBoth(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageSetPixel(img, 0, 0, red)
	ImageFlip(img, ImgFlipBoth)
	if r, _, _, _ := colorAt(t, img, 3, 3); r != 255 {
		t.Error("both flip did not move pixel to opposite corner")
	}
}

func TestImageCrop(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 3, 3, 6, 6, red)

	dst := ImageCrop(img, image.Rect(3, 3, 7, 7))
	if dst == nil {
		t.Fatal("crop returned nil")
	}
	if dst.Width() != 4 || dst.Height() != 4 {
		t.Errorf("crop size = %dx%d", dst.Width(), dst.Height())
	}
	if r, _, _, _ := colorAt(t, dst, 0, 0); r != 255 {
		t.Error("cropped top-left should be red")
	}
}

func TestImageRotate90(t *testing.T) {
	img := ImageCreateTrueColor(4, 8)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageFilledRectangle(img, 0, 0, 3, 7, black)
	// Mark a tall vertical strip of red on the right
	ImageFilledRectangle(img, 3, 0, 3, 7, red)

	dst := ImageRotate(img, 90, black)
	if dst == nil {
		t.Fatal("rotate nil")
	}
	// After 90° CCW, the right column becomes the top row.
	if dst.Width() != 8 || dst.Height() != 4 {
		t.Errorf("rotated size = %dx%d", dst.Width(), dst.Height())
	}
	// Top row should be red-ish
	if r, _, _, _ := colorAt(t, dst, 4, 0); r < 200 {
		t.Errorf("expected red-ish top row, got r=%d", r)
	}
}

func TestImageRotateBackground(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 9, 9, white)

	bg := ImageColorAllocateAlpha(img, 0, 255, 0, 0)
	dst := ImageRotate(img, 45, bg)
	if dst == nil {
		t.Fatal("rotate nil")
	}
	// A corner of the rotated output should be the bg color
	_, g, _, _ := colorAt(t, dst, 0, 0)
	if g < 200 {
		t.Errorf("corner bg green = %d", g)
	}
}
