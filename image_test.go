package gogd

import (
	"image"
	"testing"
)

func TestCreateTrueColor(t *testing.T) {
	img := ImageCreateTrueColor(10, 20)
	if img == nil {
		t.Fatal("nil image")
	}
	if !ImageIsTrueColor(img) {
		t.Error("should be truecolor")
	}
	if w, h := ImageSX(img), ImageSY(img); w != 10 || h != 20 {
		t.Errorf("size = %dx%d, want 10x20", w, h)
	}
	// Default fill should be opaque black.
	r, g, b, a := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 0 || g != 0 || b != 0 || a != AlphaOpaque {
		t.Errorf("default pixel = %d,%d,%d,%d", r, g, b, a)
	}
}

func TestCreateTrueColorInvalid(t *testing.T) {
	if ImageCreateTrueColor(0, 10) != nil || ImageCreateTrueColor(10, -1) != nil {
		t.Error("expected nil for non-positive dimensions")
	}
}

func TestCreatePalette(t *testing.T) {
	img := ImageCreate(5, 5)
	if img == nil {
		t.Fatal("nil image")
	}
	if ImageIsTrueColor(img) {
		t.Error("palette image reports truecolor")
	}
	if n := ImageColorsTotal(img); n != 0 {
		t.Errorf("empty palette count = %d", n)
	}
	red := ImageColorAllocate(img, 255, 0, 0)
	if red != 0 {
		t.Errorf("first allocate index = %d, want 0", red)
	}
	if n := ImageColorsTotal(img); n != 1 {
		t.Errorf("after one allocate: %d", n)
	}
	// Every pixel should read back as the first allocated color.
	if got := ImageColorAt(img, 2, 2); got != red {
		t.Errorf("palette background = %d, want %d", got, red)
	}
}

func TestAllocateAlphaRoundtripTrueColor(t *testing.T) {
	img := ImageCreateTrueColor(1, 1)
	c := ImageColorAllocateAlpha(img, 10, 20, 30, 40)
	r, g, b, a := ImageColorsForIndex(img, c)
	if r != 10 || g != 20 || b != 30 || a != 40 {
		t.Errorf("roundtrip = %d,%d,%d,%d", r, g, b, a)
	}
}

func TestSetPixelTrueColorNoBlend(t *testing.T) {
	img := ImageCreateTrueColor(3, 3)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	if !ImageSetPixel(img, 1, 1, red) {
		t.Fatal("SetPixel failed")
	}
	if got := ImageColorAt(img, 1, 1); got != red {
		t.Errorf("ColorAt = 0x%08x, want 0x%08x", got, red)
	}
}

func TestSetPixelTrueColorBlend(t *testing.T) {
	img := ImageCreateTrueColor(1, 1)
	// Default alpha blending ON. Draw semi-transparent red over opaque black.
	c := ImageColorAllocateAlpha(img, 255, 0, 0, 63) // ~half-transparent
	if !ImageSetPixel(img, 0, 0, c) {
		t.Fatal("SetPixel failed")
	}
	r, g, b, a := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r == 0 || r == 255 {
		t.Errorf("blended red = %d (expected between 0 and 255)", r)
	}
	if g != 0 || b != 0 {
		t.Errorf("unexpected channels g=%d b=%d", g, b)
	}
	if a != AlphaOpaque {
		t.Errorf("blend result should be opaque, got a=%d", a)
	}
}

func TestSetPixelPalette(t *testing.T) {
	img := ImageCreate(3, 3)
	ImageColorAllocate(img, 0, 0, 0)
	red := ImageColorAllocate(img, 255, 0, 0)
	if !ImageSetPixel(img, 2, 2, red) {
		t.Fatal("SetPixel failed")
	}
	if got := ImageColorAt(img, 2, 2); got != red {
		t.Errorf("ColorAt = %d, want %d", got, red)
	}
}

func TestSetPixelOutOfBounds(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	if ImageSetPixel(img, -1, 0, 0) {
		t.Error("expected false for negative x")
	}
	if ImageSetPixel(img, 0, 2, 0) {
		t.Error("expected false for y == height")
	}
	if ImageColorAt(img, 5, 5) != ColorNone {
		t.Error("expected ColorNone for oob read")
	}
}

func TestColorClosestHWB(t *testing.T) {
	img := ImageCreate(1, 1)
	ImageColorAllocate(img, 0, 0, 0)       // 0: black
	ImageColorAllocate(img, 255, 255, 255) // 1: white
	ImageColorAllocate(img, 255, 0, 0)     // 2: red
	ImageColorAllocate(img, 0, 255, 0)     // 3: green

	if got := ImageColorClosestHWB(img, 250, 10, 10); got != 2 {
		t.Errorf("near-red HWB = %d, want 2", got)
	}
	if got := ImageColorClosestHWB(img, 10, 10, 10); got != 0 {
		t.Errorf("near-black HWB = %d, want 0", got)
	}
}

func TestColorMatch(t *testing.T) {
	// palette image with two indices
	pal := ImageCreate(4, 4)
	black := ImageColorAllocate(pal, 0, 0, 0)
	red := ImageColorAllocate(pal, 200, 0, 0)
	ImageFilledRectangle(pal, 0, 0, 1, 3, black)
	ImageFilledRectangle(pal, 2, 0, 3, 3, red)

	// truecolor "source of truth" with nearby but different shades.
	tc := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(tc, false)
	dim := ImageColorAllocate(tc, 10, 20, 30)
	bright := ImageColorAllocate(tc, 255, 100, 50)
	ImageFilledRectangle(tc, 0, 0, 1, 3, dim)
	ImageFilledRectangle(tc, 2, 0, 3, 3, bright)

	if !ImageColorMatch(pal, tc) {
		t.Fatal("colormatch failed")
	}
	r, g, b, _ := ImageColorsForIndex(pal, black)
	if r != 10 || g != 20 || b != 30 {
		t.Errorf("black slot not updated: %d,%d,%d", r, g, b)
	}
	r, g, b, _ = ImageColorsForIndex(pal, red)
	if r != 255 || g != 100 || b != 50 {
		t.Errorf("red slot not updated: %d,%d,%d", r, g, b)
	}
}

func TestColorMatchBoundsMismatch(t *testing.T) {
	pal := ImageCreate(2, 2)
	ImageColorAllocate(pal, 0, 0, 0)
	tc := ImageCreateTrueColor(3, 3)
	if ImageColorMatch(pal, tc) {
		t.Error("expected false on bounds mismatch")
	}
}

func TestColorExactAndClosest(t *testing.T) {
	img := ImageCreate(1, 1)
	ImageColorAllocate(img, 0, 0, 0)
	ImageColorAllocate(img, 255, 0, 0)
	ImageColorAllocate(img, 0, 255, 0)
	if got := ImageColorExact(img, 255, 0, 0); got != 1 {
		t.Errorf("exact red index = %d, want 1", got)
	}
	if got := ImageColorExact(img, 10, 10, 10); got != ColorNone {
		t.Errorf("exact miss = %d, want %d", got, ColorNone)
	}
	if got := ImageColorClosest(img, 250, 0, 0); got != 1 {
		t.Errorf("closest to red = %d, want 1", got)
	}
}

func TestColorResolveFallsBackToClosestWhenFull(t *testing.T) {
	img := ImageCreate(1, 1)
	for i := 0; i < 256; i++ {
		if ImageColorAllocate(img, i, 0, 0) == ColorNone {
			t.Fatalf("allocate %d failed", i)
		}
	}
	if ImageColorAllocate(img, 0, 0, 255) != ColorNone {
		t.Error("expected ColorNone on 257th allocate")
	}
	// Resolve with a color not in palette -> closest.
	got := ImageColorResolve(img, 128, 10, 10)
	if got == ColorNone {
		t.Error("resolve should fall back to closest")
	}
}

func TestTransparentGetSet(t *testing.T) {
	img := ImageCreateTrueColor(1, 1)
	if got := ImageColorTransparent(img, -1); got != ColorNone {
		t.Errorf("initial transparent = %d, want %d", got, ColorNone)
	}
	ImageColorTransparent(img, 42)
	if got := ImageColorTransparent(img, -1); got != 42 {
		t.Errorf("after set = %d, want 42", got)
	}
}

func TestImageSatisfiesImageImage(t *testing.T) {
	var _ image.Image = (*Image)(nil)
	img := ImageCreateTrueColor(2, 2)
	if img.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Errorf("Bounds = %v", img.Bounds())
	}
	if img.At(0, 0) == nil {
		t.Error("At returned nil")
	}
}

func TestGDAlphaRoundtrip(t *testing.T) {
	for _, a := range []int{0, 1, 32, 63, 64, 100, 127} {
		back := stdAlphaToGD(gdAlphaToStdAlpha(a))
		if testAbs(back-a) > 1 {
			t.Errorf("roundtrip alpha %d -> %d", a, back)
		}
	}
}

func testAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
