package gogd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestImageFontDims(t *testing.T) {
	if ImageFontWidth(3) != 7 || ImageFontHeight(3) != 13 {
		t.Error("font 3 dims wrong")
	}
	if ImageFontWidth(1) != 5 || ImageFontHeight(1) != 8 {
		t.Error("font 1 dims wrong")
	}
	if ImageFontWidth(5) != 9 || ImageFontHeight(5) != 15 {
		t.Error("font 5 dims wrong")
	}
}

func TestImageStringDrawsSomething(t *testing.T) {
	img := ImageCreateTrueColor(100, 20)
	ImageAlphaBlending(img, false)
	bg := ImageColorAllocate(img, 0, 0, 0)
	ImageFilledRectangle(img, 0, 0, 99, 19, bg)
	red := ImageColorAllocate(img, 255, 0, 0)
	if !ImageString(img, 3, 2, 2, "Hi", red) {
		t.Fatal("ImageString failed")
	}
	// Scan for any non-black pixel: text should have written something.
	found := false
	for y := 0; y < 20 && !found; y++ {
		for x := 0; x < 100 && !found; x++ {
			r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, x, y))
			if r > 128 {
				found = true
			}
		}
	}
	if !found {
		t.Error("no red pixels rendered")
	}
}

func TestImageStringUp(t *testing.T) {
	img := ImageCreateTrueColor(40, 40)
	ImageAlphaBlending(img, false)
	bg := ImageColorAllocate(img, 0, 0, 0)
	ImageFilledRectangle(img, 0, 0, 39, 39, bg)
	red := ImageColorAllocate(img, 255, 0, 0)
	if !ImageStringUp(img, 3, 5, 35, "A", red) {
		t.Fatal("StringUp failed")
	}
	// Look for any red pixel.
	found := false
	for y := 0; y < 40 && !found; y++ {
		for x := 0; x < 40 && !found; x++ {
			r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, x, y))
			if r > 128 {
				found = true
			}
		}
	}
	if !found {
		t.Error("ImageStringUp produced no red pixels")
	}
}

func TestImageTTFBBox(t *testing.T) {
	path := writeGoRegular(t)
	box, err := ImageTTFBBox(24, 0, path, "Abc")
	if err != nil {
		t.Fatalf("bbox: %v", err)
	}
	// Lower-right X should be > 0 (text has width), upper-right Y should be < 0 (ascender).
	if box[2] <= 0 {
		t.Errorf("bbox width = %d", box[2])
	}
	if box[5] >= 0 {
		t.Errorf("bbox upper Y = %d (expected < 0)", box[5])
	}
}

func TestImageTTFText(t *testing.T) {
	path := writeGoRegular(t)
	img := ImageCreateTrueColor(120, 40)
	ImageAlphaBlending(img, false)
	bg := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 119, 39, bg)
	black := ImageColorAllocate(img, 0, 0, 0)
	_, err := ImageTTFText(img, 24, 0, 5, 30, black, path, "Go")
	if err != nil {
		t.Fatalf("draw: %v", err)
	}
	// Sample the image — should contain some black pixels from the glyphs.
	found := false
	for y := 0; y < 40 && !found; y++ {
		for x := 0; x < 120 && !found; x++ {
			r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, x, y))
			if r+g+b < 60 {
				found = true
			}
		}
	}
	if !found {
		t.Error("TTF drew no dark pixels")
	}
}

func TestImageTTFTextAngle(t *testing.T) {
	path := writeGoRegular(t)
	img := ImageCreateTrueColor(120, 120)
	ImageAlphaBlending(img, false)
	bg := ImageColorAllocate(img, 255, 255, 255)
	ImageFilledRectangle(img, 0, 0, 119, 119, bg)
	black := ImageColorAllocate(img, 0, 0, 0)
	_, err := ImageTTFText(img, 24, 45, 30, 60, black, path, "Go")
	if err != nil {
		t.Fatalf("draw rotated: %v", err)
	}
	// Just verify that the call succeeded and produced output somewhere.
	found := false
	for y := 0; y < 120 && !found; y++ {
		for x := 0; x < 120 && !found; x++ {
			r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, x, y))
			if r+g+b < 60 {
				found = true
			}
		}
	}
	if !found {
		t.Error("rotated TTF drew no dark pixels")
	}
}

func TestImageTTFMissingFont(t *testing.T) {
	_, err := ImageTTFBBox(12, 0, "/no/such/font.ttf", "x")
	if err == nil {
		t.Error("expected error for missing font")
	}
}

func writeGoRegular(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "go.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0o644); err != nil {
		t.Fatalf("write font: %v", err)
	}
	// Sanity check: the bytes we wrote are the TTF we expect.
	if !bytes.Equal(readFile(t, path), goregular.TTF) {
		t.Fatal("font bytes mismatch")
	}
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return b
}
