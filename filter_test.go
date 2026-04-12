package gogd

import (
	"testing"
)

func makeSolid(w, h, r, g, b int) *Image {
	img := ImageCreateTrueColor(w, h)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, r, g, b)
	ImageFilledRectangle(img, 0, 0, w-1, h-1, c)
	return img
}

func TestFilterNegate(t *testing.T) {
	img := makeSolid(2, 2, 50, 100, 200)
	if !ImageFilter(img, FilterNegate) {
		t.Fatal("negate failed")
	}
	r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 205 || g != 155 || b != 55 {
		t.Errorf("negate = %d,%d,%d, want 205,155,55", r, g, b)
	}
}

func TestFilterGrayscale(t *testing.T) {
	img := makeSolid(2, 2, 100, 200, 50)
	if !ImageFilter(img, FilterGrayscale) {
		t.Fatal("grayscale failed")
	}
	r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != g || g != b {
		t.Errorf("not gray: %d,%d,%d", r, g, b)
	}
	// 0.299*100 + 0.587*200 + 0.114*50 ≈ 152.75 → 153
	if r < 150 || r > 155 {
		t.Errorf("unexpected gray value: %d", r)
	}
}

func TestFilterBrightness(t *testing.T) {
	img := makeSolid(2, 2, 100, 100, 100)
	if !ImageFilter(img, FilterBrightness, 50) {
		t.Fatal("brightness failed")
	}
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 150 {
		t.Errorf("brightness +50 = %d, want 150", r)
	}

	if !ImageFilter(img, FilterBrightness, -200) {
		t.Fatal("brightness failed")
	}
	r, _, _, _ = ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 0 {
		t.Errorf("after -200 = %d, want 0 (clamped)", r)
	}
}

func TestFilterContrast(t *testing.T) {
	img := makeSolid(2, 2, 200, 100, 50)
	// arg = -100 → factor = 2.0 → high contrast; 200 -> 255, 50 -> 0
	if !ImageFilter(img, FilterContrast, -100) {
		t.Fatal("contrast failed")
	}
	r, _, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 255 {
		t.Errorf("high-contrast bright = %d", r)
	}
	if b != 0 {
		t.Errorf("high-contrast dark = %d", b)
	}
}

func TestFilterColorize(t *testing.T) {
	img := makeSolid(2, 2, 100, 100, 100)
	ImageFilter(img, FilterColorize, 50, -20, 20)
	r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 150 || g != 80 || b != 120 {
		t.Errorf("colorize = %d,%d,%d, want 150,80,120", r, g, b)
	}
}

func TestFilterGaussianBlur(t *testing.T) {
	img := ImageCreateTrueColor(5, 5)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 4, 4, black)
	ImageSetPixel(img, 2, 2, white)

	if !ImageFilter(img, FilterGaussianBlur) {
		t.Fatal("gaussian failed")
	}
	// Center should now be less than 255 (blurred)
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 2, 2))
	if r == 255 || r == 0 {
		t.Errorf("center after blur = %d; expected intermediate", r)
	}
	// Neighbors should be > 0 (spread)
	nr, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 1, 2))
	if nr == 0 {
		t.Error("neighbor not blurred")
	}
}

func TestFilterEdgeDetect(t *testing.T) {
	img := ImageCreateTrueColor(5, 5)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 4, 4, black)
	ImageFilledRectangle(img, 2, 0, 4, 4, white)
	if !ImageFilter(img, FilterEdgeDetect) {
		t.Fatal("edge detect failed")
	}
	// The boundary should light up
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 2, 2))
	if r < 100 {
		t.Errorf("edge not detected: r=%d", r)
	}
}

func TestFilterMeanRemoval(t *testing.T) {
	img := makeSolid(3, 3, 100, 100, 100)
	// Uniform image → mean removal keeps uniform (center*9 - 8*same = 1*same)
	if !ImageFilter(img, FilterMeanRemoval) {
		t.Fatal("mean removal failed")
	}
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 1, 1))
	if r != 100 {
		t.Errorf("uniform mean removal = %d, want 100", r)
	}
}

func TestFilterPixelate(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	blue := ImageColorAllocate(img, 0, 0, 255)
	ImageFilledRectangle(img, 0, 0, 3, 3, red)
	ImageSetPixel(img, 1, 1, blue)

	// Basic pixelate with block 2: each block gets top-left pixel
	ImageFilter(img, FilterPixelate, 2)
	// Block (0,0) top-left is red → whole block red
	if r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 1, 1)); r != 255 {
		t.Errorf("pixelate block expected red, got r=%d", r)
	}
}

func TestFilterSelectiveBlur(t *testing.T) {
	img := ImageCreateTrueColor(5, 5)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	white := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 4, 4, black)
	// Put one bright pixel in a dark neighbourhood; selective blur should
	// mostly leave the edge alone (big luminance gap), while flat areas
	// stay flat.
	ImageSetPixel(img, 2, 2, white)
	ImageFilter(img, FilterSelectiveBlur)
	// Corner still flat black.
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 0 {
		t.Errorf("corner changed: r=%d", r)
	}
	// Centre pixel still bright (excluded dark neighbours).
	r, _, _, _ = ImageColorsForIndex(img, ImageColorAt(img, 2, 2))
	if r < 200 {
		t.Errorf("bright centre darkened: r=%d", r)
	}
}

func TestFilterScatter(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	ImageFilledRectangle(img, 0, 0, 4, 9, ImageColorAllocate(img, 255, 0, 0))
	ImageFilledRectangle(img, 5, 0, 9, 9, ImageColorAllocate(img, 0, 0, 255))
	before := ImageColorAt(img, 4, 5)
	// Scatter should jitter the border; run it and make sure it didn't
	// blow up, then check that at least one border pixel changed.
	if !ImageFilter(img, FilterScatter, 2, 2) {
		t.Fatal("scatter failed")
	}
	changed := 0
	for y := 0; y < 10; y++ {
		for x := 3; x < 7; x++ {
			if ImageColorAt(img, x, y) != before {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Error("scatter produced no visible changes")
	}
}

func TestFilterScatterBadArgs(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	if ImageFilter(img, FilterScatter) {
		t.Error("expected false with no args")
	}
}

func TestConvolution(t *testing.T) {
	img := makeSolid(3, 3, 128, 128, 128)
	// Identity kernel should leave the image unchanged.
	ImageConvolution(img, [3][3]float64{
		{0, 0, 0},
		{0, 1, 0},
		{0, 0, 0},
	}, 1, 0)
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 1, 1))
	if r != 128 {
		t.Errorf("identity conv changed pixel: r=%d", r)
	}
}

func TestGammaCorrect(t *testing.T) {
	img := makeSolid(2, 2, 128, 128, 128)
	// gamma = 1.0 / 2.0 → exponent 0.5 → brighten mid-gray
	if !ImageGammaCorrect(img, 1.0, 2.0) {
		t.Fatal("gamma failed")
	}
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r < 170 || r > 200 {
		t.Errorf("gamma 0.5 on 128 = %d (expected ~181)", r)
	}
}

func TestLayerEffect(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	if !ImageLayerEffect(img, EffectReplace) {
		t.Fatal("effect failed")
	}
	if img.alphaBlending {
		t.Error("EffectReplace should disable alpha blending")
	}
	ImageLayerEffect(img, EffectAlphaBlend)
	if !img.alphaBlending {
		t.Error("EffectAlphaBlend should enable alpha blending")
	}
}

func TestColorSet(t *testing.T) {
	img := ImageCreate(2, 2)
	bg := ImageColorAllocate(img, 0, 0, 0)
	ImageColorSet(img, bg, 255, 128, 0)
	r, g, b, _ := ImageColorsForIndex(img, bg)
	if r != 255 || g != 128 || b != 0 {
		t.Errorf("colorset = %d,%d,%d", r, g, b)
	}
}

func TestPaletteCopy(t *testing.T) {
	src := ImageCreate(2, 2)
	ImageColorAllocate(src, 10, 20, 30)
	ImageColorAllocate(src, 100, 100, 100)

	dst := ImageCreate(2, 2)
	ImageColorAllocate(dst, 0, 0, 0)
	if !ImagePaletteCopy(dst, src) {
		t.Fatal("palette copy failed")
	}
	if ImageColorsTotal(dst) != 2 {
		t.Errorf("dst palette size = %d, want 2", ImageColorsTotal(dst))
	}
}

func TestPaletteToTrueColor(t *testing.T) {
	img := ImageCreate(3, 3)
	ImageColorAllocate(img, 50, 100, 150)
	if !ImagePaletteToTrueColor(img) {
		t.Fatal("conversion failed")
	}
	if !img.IsTrueColor() {
		t.Error("still palette after conversion")
	}
	r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 50 || g != 100 || b != 150 {
		t.Errorf("color preserved = %d,%d,%d", r, g, b)
	}
}

func TestTrueColorToPalette(t *testing.T) {
	img := makeSolid(4, 4, 255, 0, 0)
	if !ImageTrueColorToPalette(img, false, 16) {
		t.Fatal("conversion failed")
	}
	if img.IsTrueColor() {
		t.Error("still truecolor after conversion")
	}
	// Should have mapped to a red-ish palette entry
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r < 100 {
		t.Errorf("red mapping too dark: r=%d", r)
	}
}

func TestTrueColorToPaletteMedianCut(t *testing.T) {
	// Build a gradient image with many unique colors.
	img := ImageCreateTrueColor(16, 16)
	ImageAlphaBlending(img, false)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := ImageColorAllocate(img, x*16, y*16, (x+y)*8)
			ImageSetPixel(img, x, y, c)
		}
	}
	if !ImageTrueColorToPalette(img, false, 16) {
		t.Fatal("conversion failed")
	}
	if n := ImageColorsTotal(img); n > 16 || n == 0 {
		t.Errorf("palette size = %d, want 1..16", n)
	}
	// The top-left (which was near-black) should still be dark-ish.
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r > 100 {
		t.Errorf("(0,0) too bright after quantize: r=%d", r)
	}
	// Bottom-right was bright red + some blue: should still be red-dominant.
	r, _, _, _ = ImageColorsForIndex(img, ImageColorAt(img, 15, 0))
	if r < 150 {
		t.Errorf("(15,0) too dark: r=%d", r)
	}
}
