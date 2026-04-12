package gogd

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// Tests covering the newly-opened API: gogd drawing on plain stdlib
// image types without wrapping them in a *gogd.Image first.

func TestSetPixelOnNRGBA(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	red := packGDColor(255, 0, 0, 0)
	if !ImageSetPixel(m, 1, 1, Color(red)) {
		t.Fatal("SetPixel on *image.NRGBA failed")
	}
	if got := m.NRGBAAt(1, 1); got != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("pixel = %+v", got)
	}
	if c := ImageColorAt(m, 1, 1); c != Color(red) {
		t.Errorf("ColorAt = 0x%x, want 0x%x", c, red)
	}
}

func TestSetPixelOnRGBA(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 4, 4))
	col := Color(packGDColor(0, 255, 0, 0))
	if !ImageSetPixel(m, 2, 2, col) {
		t.Fatal("SetPixel on *image.RGBA failed")
	}
	r, g, b, a := m.RGBAAt(2, 2).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Errorf("pixel = %d,%d,%d,%d", r, g, b, a)
	}
}

func TestSetPixelOnPaletted(t *testing.T) {
	p := color.Palette{color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 0, 0, 255}}
	m := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	// Color 1 = red in this palette.
	if !ImageSetPixel(m, 0, 0, 1) {
		t.Fatal("SetPixel on *image.Paletted failed")
	}
	if got := m.ColorIndexAt(0, 0); got != 1 {
		t.Errorf("index = %d", got)
	}
	if c := ImageColorAt(m, 0, 0); c != 1 {
		t.Errorf("ColorAt = %d, want 1", c)
	}
}

func TestImageLineOnNRGBA(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	if !ImageLine(m, 0, 0, 9, 9, Color(packGDColor(255, 255, 0, 0))) {
		t.Fatal("ImageLine failed")
	}
	// Diagonal should be yellow.
	got := m.NRGBAAt(5, 5)
	if got.R != 255 || got.G != 255 {
		t.Errorf("(5,5) = %+v", got)
	}
}

func TestImageFilledRectangleOnRGBA(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if !ImageFilledRectangle(m, 2, 2, 5, 5, Color(packGDColor(0, 0, 255, 0))) {
		t.Fatal("FilledRectangle failed")
	}
	got := m.RGBAAt(3, 3)
	if got.B != 0xff {
		t.Errorf("interior not blue: %+v", got)
	}
	out := m.RGBAAt(0, 0)
	if out.B != 0 {
		t.Errorf("outside touched: %+v", out)
	}
}

func TestImageFilterOnNRGBA(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			m.SetNRGBA(x, y, color.NRGBA{100, 100, 100, 255})
		}
	}
	if !ImageFilter(m, FilterNegate) {
		t.Fatal("filter failed")
	}
	got := m.NRGBAAt(1, 1)
	if got.R != 155 {
		t.Errorf("negate mid: %+v", got)
	}
}

func TestImageEncodePlainNRGBA(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			m.SetNRGBA(x, y, color.NRGBA{255, 128, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := ImagePNG(m, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("empty PNG")
	}
}

func TestImageCopyBetweenTypes(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetNRGBA(x, y, color.NRGBA{0, 200, 0, 255})
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if !ImageCopy(dst, src, 3, 3, 0, 0, 4, 4) {
		t.Fatal("copy failed")
	}
	got := dst.RGBAAt(5, 5)
	if got.G < 100 {
		t.Errorf("copied green = %+v", got)
	}
}

func TestImageFlipOnRGBA(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 4, 4))
	m.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	if !ImageFlip(m, ImgFlipHorizontal) {
		t.Fatal("flip failed")
	}
	got := m.RGBAAt(3, 0)
	if got.R != 255 {
		t.Errorf("flipped (3,0) = %+v", got)
	}
	if m.RGBAAt(0, 0).R == 255 {
		t.Error("original pixel still at (0,0)")
	}
}

func TestImageCropOnNRGBA(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 3; y < 7; y++ {
		for x := 3; x < 7; x++ {
			src.SetNRGBA(x, y, color.NRGBA{255, 0, 0, 255})
		}
	}
	dst := ImageCrop(src, image.Rect(3, 3, 7, 7))
	if dst == nil {
		t.Fatal("crop nil")
	}
	if dst.Width() != 4 || dst.Height() != 4 {
		t.Errorf("crop = %dx%d", dst.Width(), dst.Height())
	}
	r, _, _, _ := ImageColorsForIndex(dst, ImageColorAt(dst, 0, 0))
	if r != 255 {
		t.Error("cropped first pixel not red")
	}
}

func TestImageSXOnStdlibImage(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 42, 7))
	if ImageSX(m) != 42 || ImageSY(m) != 7 {
		t.Errorf("size on *image.RGBA = %dx%d", ImageSX(m), ImageSY(m))
	}
	if ImageIsTrueColor(m) != true {
		t.Error("RGBA should be reported as truecolor")
	}
	pm := image.NewPaletted(image.Rect(0, 0, 3, 3), color.Palette{color.Black})
	if ImageIsTrueColor(pm) {
		t.Error("Paletted should be reported as palette")
	}
}
